package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/fstx"
	"github.com/Zxilly/cjv/internal/i18n"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/Zxilly/cjv/internal/utils"
	"github.com/fatih/color"
)

// Options carries the small adapter surface the lifecycle module needs from
// callers. The install implementation is shared by CLI commands and proxy
// auto-install; presentation stays outside the core module.
type Options struct {
	IsJSON               func() bool
	EnsurePathConfigured func()
	ComponentInstall     func(context.Context, component.Roots, toolchain.ToolchainName, component.Name, string, string, bool) error
	EnsureManagedBinary  func() (string, error)
	CreateProxyLinks     func() error
	ValidateInstallation func(dir, tuple string) error
}

func (o Options) json() bool {
	return o.IsJSON != nil && o.IsJSON()
}

func (o Options) note(s string) {
	if !o.json() {
		fmt.Println(s)
	}
}

func (o Options) green(key string, data i18n.MsgData) {
	if !o.json() {
		color.Green(i18n.T(key, data))
	}
}

func (o Options) ensurePathConfigured() {
	if o.EnsurePathConfigured != nil {
		o.EnsurePathConfigured()
		return
	}
	EnsurePathConfigured()
}

func (o Options) installComponent(ctx context.Context, roots component.Roots, tc toolchain.ToolchainName, name component.Name, tuple, downloadsDir string, force bool) error {
	install := o.ComponentInstall
	if install == nil {
		install = component.Install
	}
	return install(ctx, roots, tc, name, tuple, downloadsDir, force)
}

func (o Options) createProxyLinks() error {
	if o.CreateProxyLinks == nil {
		return nil
	}
	return o.CreateProxyLinks()
}

func (o Options) ensureManagedBinary() error {
	if o.EnsureManagedBinary == nil {
		return nil
	}
	_, err := o.EnsureManagedBinary()
	return err
}

func (o Options) validateInstallation(dir, tuple string) error {
	if o.ValidateInstallation != nil {
		return o.ValidateInstallation(dir, tuple)
	}
	return validateInstallation(dir, tuple)
}

// InstallToolchainWithOptions installs a toolchain with optional force re-install.
func InstallToolchainWithOptions(ctx context.Context, input string, force bool, opts Options) error {
	return InstallToolchainWithExtras(ctx, input, nil, nil, force, opts)
}

// InstallToolchainWithTargets installs the host toolchain plus optional cross SDK target variants.
func InstallToolchainWithTargets(ctx context.Context, input string, targets []string, force bool, opts Options) error {
	return InstallToolchainWithExtras(ctx, input, targets, nil, force, opts)
}

// InstallToolchainWithExtras installs the host toolchain plus optional cross
// SDK target variants and optional components.
func InstallToolchainWithExtras(ctx context.Context, input string, targets, components []string, force bool, opts Options) error {
	if ctx == nil {
		ctx = context.Background()
	}
	name, err := toolchain.ParseToolchainName(input)
	if err != nil {
		return err
	}
	if name.IsCustom() {
		return errors.New(i18n.T("InstallCustomToolchain", i18n.MsgData{"Name": input}))
	}

	sf, settings, err := LoadSettings()
	if err != nil {
		return err
	}

	normalizedTargets, err := sdktarget.NormalizeList(targets)
	if err != nil {
		return err
	}
	if name.Target != "" && len(normalizedTargets) > 0 {
		return fmt.Errorf("cannot combine target variant toolchain name %q with --target; pass the host toolchain name and --target instead", input)
	}

	fetcher := NewManifestFetcher(settings.ManifestURL, opts)

	resolved, err := ResolveAndLocate(ctx, name, settings, fetcher)
	if err != nil {
		return err
	}

	if name.Target != "" {
		if err := InstallResolvedNoDefault(ctx, resolved, settings, sf, force, opts); err != nil {
			return err
		}
	} else if err := InstallResolved(ctx, resolved, settings, sf, force, opts); err != nil {
		return err
	}

	targetBase := name
	if len(normalizedTargets) > 0 {
		hostResolved, err := toolchain.ParseToolchainName(resolved.Name)
		if err != nil {
			return err
		}
		targetBase = toolchain.ToolchainName{Channel: hostResolved.Channel, Version: hostResolved.Version}
	}

	var targetNames []string
	for _, target := range normalizedTargets {
		resolvedTarget, err := ResolveAndLocateWithTarget(ctx, targetBase, settings, fetcher, target)
		if err != nil {
			return err
		}
		if err := InstallResolvedNoDefault(ctx, resolvedTarget, settings, sf, force, opts); err != nil {
			return err
		}
		targetNames = append(targetNames, resolvedTarget.Name)
	}

	if len(components) > 0 {
		if len(normalizedTargets) > 0 {
			for _, targetName := range targetNames {
				if err := InstallComponentsList(ctx, targetName, components, force, false, opts); err != nil {
					return err
				}
			}
		} else if err := InstallComponentsList(ctx, resolved.Name, components, force, false, opts); err != nil {
			return err
		}
	}
	return nil
}

// LoadSettings loads the cached user settings file used by lifecycle operations.
func LoadSettings() (*config.SettingsFile, *config.Settings, error) {
	sf, err := config.DefaultSettingsFile()
	if err != nil {
		return nil, nil, err
	}
	settings, err := sf.Load()
	if err != nil {
		return nil, nil, err
	}
	return sf, settings, nil
}

// ManifestFetcher fetches the SDK manifest at most once per lifecycle operation.
type ManifestFetcher struct {
	once sync.Once
	url  string
	opts Options
	m    *dist.Manifest
	err  error
}

func NewManifestFetcher(url string, opts Options) *ManifestFetcher {
	return &ManifestFetcher{url: url, opts: opts}
}

func (f *ManifestFetcher) Get(ctx context.Context) (*dist.Manifest, error) {
	f.once.Do(func() {
		f.opts.note(i18n.T("FetchingManifest", nil))
		f.m, f.err = FetchManifest(ctx, f.url)
	})
	return f.m, f.err
}

// InstallComponentsForToolchain backs the proxy auto_install path: it resolves
// tcInput to an already-installed toolchain and installs missing components quietly.
func InstallComponentsForToolchain(ctx context.Context, tcInput string, components []string, opts Options) error {
	if len(components) == 0 {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	name, err := toolchain.ParseToolchainName(tcInput)
	if err != nil {
		return err
	}
	installedDir, err := toolchain.FindInstalled(name)
	if err != nil {
		return err
	}
	return InstallComponentsList(ctx, filepath.Base(installedDir), components, false, true, opts)
}

// InstallComponentsList expects resolvedName as "<channel>-<version>".
func InstallComponentsList(ctx context.Context, resolvedName string, components []string, force, quiet bool, opts Options) error {
	resolvedTC, err := toolchain.ParseToolchainName(resolvedName)
	if err != nil {
		return err
	}
	if resolvedTC.IsCustom() {
		return &cjverr.ComponentRequiresHostError{Component: strings.Join(components, ", ")}
	}
	parsed, err := component.NormalizeList(components)
	if err != nil {
		return err
	}
	_, settings, err := LoadSettings()
	if err != nil {
		return err
	}
	tuple := resolvedTC.Target
	if tuple == "" {
		tuple, err = dist.CurrentHostTuple(settings.DefaultHost)
		if err != nil {
			return err
		}
	}
	downloadsDir, err := config.DownloadsDir()
	if err != nil {
		return err
	}
	roots, err := component.RootsFor(resolvedName)
	if err != nil {
		return err
	}
	snap, err := component.TakeSnapshot(roots, parsed)
	if err != nil {
		return err
	}
	defer snap.Cleanup() //nolint:errcheck
	for _, c := range parsed {
		if err := opts.installComponent(ctx, roots, resolvedTC, c, tuple, downloadsDir, force); err != nil {
			var alreadyErr *cjverr.ComponentAlreadyInstalledError
			if errors.As(err, &alreadyErr) {
				if !quiet && !opts.json() {
					fmt.Println(err)
				}
				continue
			}
			_ = snap.Restore() //nolint:errcheck
			return err
		}
		if !quiet {
			opts.green("ComponentInstalled", i18n.MsgData{"Toolchain": resolvedName, "Component": string(c)})
		}
	}
	return nil
}

// ResolvedToolchain holds the result of toolchain resolution.
type ResolvedToolchain struct {
	Name        string
	URL         string
	SHA256      string
	ArchiveName string
	Tuple       string
}

func InstallResolved(ctx context.Context, rt ResolvedToolchain, settings *config.Settings, sf *config.SettingsFile, force bool, opts Options) error {
	return installResolvedWithDefault(ctx, rt, settings, sf, force, true, opts)
}

func InstallResolvedNoDefault(ctx context.Context, rt ResolvedToolchain, settings *config.Settings, sf *config.SettingsFile, force bool, opts Options) error {
	return installResolvedWithDefault(ctx, rt, settings, sf, force, false, opts)
}

func installResolvedWithDefault(ctx context.Context, rt ResolvedToolchain, settings *config.Settings, sf *config.SettingsFile, force bool, allowDefault bool, opts Options) (retErr error) {
	resolvedName := rt.Name
	tcDir, err := config.ToolchainsDir()
	if err != nil {
		return err
	}
	destDir := filepath.Join(tcDir, resolvedName)
	isReinstall := false
	if _, err := os.Stat(destDir); err == nil {
		if !force {
			if opts.json() {
				return &cjverr.ToolchainAlreadyInstalledError{Name: resolvedName}
			}
			fmt.Println(i18n.T("ToolchainAlreadyInstalled", i18n.MsgData{"Name": resolvedName}))
			return nil
		}
		isReinstall = true
	}

	if err := config.EnsureDirs(); err != nil {
		return err
	}

	downloadsDir, err := config.DownloadsDir()
	if err != nil {
		return err
	}
	if u, err := url.Parse(rt.URL); err != nil || u.Path == "" {
		return fmt.Errorf("invalid toolchain download URL: %s", rt.URL)
	}

	archivePath, err := dist.DownloadCachedWithName(ctx, rt.URL, rt.SHA256, downloadsDir, rt.ArchiveName)
	if err != nil {
		return err
	}
	defer func() {
		if retErr == nil {
			_ = dist.CleanupDownload(archivePath) //nolint:errcheck
		}
	}()

	stagingDir := destDir + toolchain.StagingSuffix
	if err := utils.RemoveAllRetry(stagingDir); err != nil {
		return fmt.Errorf("failed to clean staging directory: %w", err)
	}
	defer func() {
		if retErr != nil {
			_ = utils.RemoveAllRetry(stagingDir) //nolint:errcheck
		}
	}()

	opts.note(i18n.T("Extracting", nil))
	if err := dist.InstallSDK(ctx, archivePath, stagingDir); err != nil {
		return err
	}
	if err := opts.validateInstallation(stagingDir, rt.Tuple); err != nil {
		return err
	}

	isFirstInstall := allowDefault && (settings.DefaultToolchain == "" || !defaultToolchainExists(settings.DefaultToolchain))
	if err := swapInstalledToolchain(stagingDir, destDir, isReinstall, func() error {
		if err := opts.ensureManagedBinary(); err != nil {
			return err
		}
		if err := opts.createProxyLinks(); err != nil {
			return err
		}
		if isFirstInstall {
			settings.DefaultToolchain = resolvedName
			if err := sf.Save(settings); err != nil {
				return err
			}
			opts.ensurePathConfigured()
		}
		return nil
	}); err != nil {
		return err
	}

	opts.green("ToolchainInstalled", i18n.MsgData{"Name": resolvedName})
	return nil
}

func ResolveAndLocate(ctx context.Context, name toolchain.ToolchainName, settings *config.Settings, fetcher *ManifestFetcher) (ResolvedToolchain, error) {
	return ResolveAndLocateWithTarget(ctx, name, settings, fetcher, "")
}

func ResolveAndLocateWithTarget(ctx context.Context, name toolchain.ToolchainName, settings *config.Settings, fetcher *ManifestFetcher, target string) (ResolvedToolchain, error) {
	tuple := name.Target
	if tuple == "" {
		var err error
		tuple, err = dist.CurrentTargetTuple(settings.DefaultHost, target)
		if err != nil {
			return ResolvedToolchain{}, err
		}
	}
	return ResolveAndLocateWithTuple(ctx, name, settings, fetcher, tuple)
}

func ResolveAndLocateWithTuple(ctx context.Context, name toolchain.ToolchainName, settings *config.Settings, fetcher *ManifestFetcher, tuple string) (ResolvedToolchain, error) {
	if tuple == "" {
		var err error
		tuple, err = dist.CurrentHostTuple(settings.DefaultHost)
		if err != nil {
			return ResolvedToolchain{}, err
		}
	}
	if name.Channel == toolchain.Nightly {
		return resolveNightlyWithTuple(ctx, name, settings, tuple, fetcher.opts)
	}

	manifest, err := fetcher.Get(ctx)
	if err != nil {
		return ResolvedToolchain{}, err
	}

	channel := name.Channel
	version := name.Version
	if channel == toolchain.UnknownChannel {
		found, err := manifest.FindVersionChannel(version)
		if err != nil {
			return ResolvedToolchain{}, err
		}
		channel = found
	}
	if version == "" {
		v, err := latestVersionForTuple(manifest, channel, tuple)
		if err != nil {
			return ResolvedToolchain{}, err
		}
		version = v
	}

	resolved := toolchain.ToolchainName{Channel: channel, Version: version}
	if id, err := sdktarget.ParseIdentity(tuple); err == nil && id.IsTargetVariant() {
		resolved.Target = tuple
	}
	info, err := manifest.GetDownloadInfo(channel, version, tuple)
	if err != nil {
		return ResolvedToolchain{}, err
	}
	return ResolvedToolchain{Name: resolved.String(), URL: info.URL, SHA256: info.SHA256, ArchiveName: info.Name, Tuple: tuple}, nil
}

func latestVersionForTuple(manifest *dist.Manifest, channel toolchain.Channel, tuple string) (string, error) {
	if tuple == "" {
		return manifest.GetLatestVersion(channel)
	}
	versions, err := manifest.ListVersions(channel, tuple)
	if err != nil {
		return "", err
	}
	if len(versions) > 0 {
		return versions[0], nil
	}
	latest, err := manifest.GetLatestVersion(channel)
	if err != nil {
		return "", err
	}
	return "", &cjverr.VersionNotAvailableError{Version: latest, Target: tuple}
}

// FetchNightlySHA256 is a package-level seam for tests that resolve nightly toolchains.
var FetchNightlySHA256 = dist.FetchNightlySHA256

func resolveNightlyWithTuple(ctx context.Context, name toolchain.ToolchainName, settings *config.Settings, tuple string, opts Options) (ResolvedToolchain, error) {
	if tuple == "" {
		var err error
		tuple, err = dist.CurrentHostTuple(settings.DefaultHost)
		if err != nil {
			return ResolvedToolchain{}, err
		}
	}
	version := name.Version
	if version == "" {
		opts.note(i18n.T("FetchingNightly", nil))
		v, err := dist.FetchLatestNightly(ctx, dist.DefaultNightlyAPIURL, settings.ResolveGitCodeAPIKey())
		if err != nil {
			return ResolvedToolchain{}, err
		}
		version = v
	}

	resolved := toolchain.ToolchainName{Channel: toolchain.Nightly, Version: version}
	if id, err := sdktarget.ParseIdentity(tuple); err == nil && id.IsTargetVariant() {
		resolved.Target = tuple
	}

	url, err := dist.NightlyDownloadURLForTuple(dist.DefaultNightlyBaseURL, version, tuple)
	if err != nil {
		return ResolvedToolchain{}, err
	}
	sha256, err := FetchNightlySHA256(ctx, url)
	if err != nil {
		return ResolvedToolchain{}, err
	}
	if sha256 == "" {
		opts.note(i18n.T("NightlyNoChecksum", nil))
	}
	return ResolvedToolchain{Name: resolved.String(), URL: url, SHA256: sha256, Tuple: tuple}, nil
}

func FetchManifest(ctx context.Context, manifestURL string) (*dist.Manifest, error) {
	u, err := url.Parse(manifestURL)
	if err != nil {
		return nil, fmt.Errorf("invalid manifest URL: %w", err)
	}
	switch u.Scheme {
	case "https":
	case "http":
		if !isLoopbackHost(u.Hostname()) && os.Getenv(config.EnvAllowInsecureManifest) != "1" {
			return nil, fmt.Errorf("refusing to fetch manifest over insecure HTTP from %q: use HTTPS, or set %s=1 to trust an internal mirror", u.Host, config.EnvAllowInsecureManifest)
		}
		slog.Warn("fetching manifest over insecure HTTP", "url", manifestURL)
	default:
		return nil, fmt.Errorf("invalid manifest URL scheme %q: only https and http are supported", u.Scheme)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create manifest request: %w", err)
	}
	resp, err := dist.HTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch manifest: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, dist.MaxResponseSize))
	if err != nil {
		return nil, err
	}
	return dist.ParseManifest(data)
}

func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback()
	}
	return false
}

func defaultToolchainExists(name string) bool {
	parsed, err := toolchain.ParseToolchainName(name)
	if err != nil {
		return false
	}
	_, err = toolchain.FindInstalled(parsed)
	return err == nil
}

func validateInstallation(dir, tuple string) error {
	binary := filepath.Join(dir, "bin", "cjc")
	if tuple != "" {
		if id, err := sdktarget.ParseIdentity(tuple); err == nil && strings.HasPrefix(id.HostTuple(), "win32-") {
			binary += ".exe"
		}
	} else if runtime.GOOS == "windows" {
		binary += ".exe"
	}
	if _, err := os.Stat(binary); err != nil {
		return fmt.Errorf("installation validation failed: %w", err)
	}
	return nil
}

func swapInstalledToolchain(stagingDir, destDir string, isReinstall bool, afterSwap func() error) (err error) {
	tx, txErr := fstx.NewTransaction(destDir)
	if txErr != nil {
		return fmt.Errorf("failed to begin install transaction: %w", txErr)
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		if rbErr := tx.Rollback(); rbErr != nil {
			err = errors.Join(err, fmt.Errorf("rollback after failed install also failed: %w", rbErr))
		}
	}()

	if isReinstall {
		if err := tx.RemoveDir(destDir); err != nil {
			return fmt.Errorf("failed to remove existing toolchain: %w", err)
		}
	}
	if err := tx.RenameFile(stagingDir, destDir); err != nil {
		return fmt.Errorf("failed to place new toolchain: %w", err)
	}
	if err := afterSwap(); err != nil {
		return fmt.Errorf("failed to finalize installation: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}

// EnsurePathConfigured is the default lifecycle hook for first install. CLI
// adapters inject the real shell/registry writer; proxy auto-install leaves
// PATH alone because cjv is already reachable.
func EnsurePathConfigured() {
}
