package cli

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/output"
	"github.com/Zxilly/cjv/internal/cli/selfmgmt"
	clisettings "github.com/Zxilly/cjv/internal/cli/settings"
	componentlib "github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/env"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/Zxilly/cjv/internal/selfupdate"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/Zxilly/cjv/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	forceInstall      bool
	installTargets    []string
	installComponents []string

	// ensurePathConfiguredFn is called during first install to add cjv's bin
	// directory to the user's PATH. Tests override this to avoid writing to
	// the real system PATH (e.g., the Windows registry).
	ensurePathConfiguredFn = ensurePathConfigured
	componentInstallFunc   = componentlib.Install

	installToolchainWithExtrasFn = InstallToolchainWithExtras
)

func init() {
	installCmd.Flags().BoolVar(&forceInstall, "force", false, i18n.T("InstallFlagForce", nil))
	installCmd.Flags().StringSliceVarP(&installTargets, "target", "t", nil, i18n.T("InstallFlagTarget", nil))
	installCmd.Flags().StringSliceVarP(&installComponents, "component", "c", nil, i18n.T("InstallFlagComponent", nil))
	rootCmd.AddCommand(installCmd)
}

var installCmd = &cobra.Command{
	Use:   "install <toolchain>",
	Short: i18n.T("InstallCmdShort", nil),
	Args:  cobra.ExactArgs(1),
	RunE:  runInstall,
}

type installResult struct {
	Toolchain  string   `json:"toolchain"`
	Targets    []string `json:"targets"`
	Components []string `json:"components"`
	Forced     bool     `json:"forced"`
}

func (r installResult) Text() string { return "" }

func runInstall(cmd *cobra.Command, args []string) error {
	selfmgmt.CheckSudoSafety()
	toolchain.CleanupStagingDirs()
	if err := InstallToolchainWithExtras(cmd.Context(), args[0], installTargets, installComponents, forceInstall); err != nil {
		return err
	}
	if !output.IsJSON() {
		return nil
	}
	return output.RenderTo(cmdOutput(cmd), installResult{
		Toolchain:  args[0],
		Targets:    installTargets,
		Components: installComponents,
		Forced:     forceInstall,
	})
}

// noteStep emits a progress/status line to stdout in text mode; in JSON
// mode it is suppressed so stdout remains a single JSON document.
func noteStep(s string) {
	if output.IsJSON() {
		return
	}
	fmt.Println(s)
}

// InstallToolchainWithOptions installs a toolchain with optional force re-install.
func InstallToolchainWithOptions(ctx context.Context, input string, force bool) error {
	return InstallToolchainWithExtras(ctx, input, nil, nil, force)
}

// InstallToolchainWithTargets installs the host toolchain plus optional cross SDK target variants.
func InstallToolchainWithTargets(ctx context.Context, input string, targets []string, force bool) error {
	return InstallToolchainWithExtras(ctx, input, targets, nil, force)
}

// InstallToolchainWithExtras installs the host toolchain plus optional cross
// SDK target variants and optional components.
func InstallToolchainWithExtras(ctx context.Context, input string, targets, components []string, force bool) error {
	if ctx == nil {
		ctx = context.Background()
	}
	name, err := toolchain.ParseToolchainName(input)
	if err != nil {
		return err
	}
	if name.IsCustom() {
		return fmt.Errorf("cannot install custom toolchain '%s': use 'cjv toolchain link' instead", input)
	}

	sf, settings, err := clisettings.LoadSettings()
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

	fetcher := newManifestFetcher(settings.ManifestURL)

	resolved, err := resolveAndLocate(ctx, name, settings, fetcher)
	if err != nil {
		return err
	}

	if name.Target != "" {
		if err := installResolvedNoDefault(ctx, resolved, settings, sf, force); err != nil {
			return err
		}
	} else {
		if err := installResolved(ctx, resolved, settings, sf, force); err != nil {
			return err
		}
	}

	// Pin cross-compile target SDKs to the host's resolved concrete version so
	// the host toolchain and its target SDKs always share a version. A
	// channel-alias / latest install otherwise resolves each target's version
	// independently (latestVersionForTuple), which can diverge from the host
	// (e.g. a lagging target build) and later break `envsetup --target`, which
	// reconstructs the target name from the host version.
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
		resolvedTarget, err := resolveAndLocateWithTarget(ctx, targetBase, settings, fetcher, target)
		if err != nil {
			return err
		}
		if err := installResolvedNoDefault(ctx, resolvedTarget, settings, sf, force); err != nil {
			return err
		}
		targetNames = append(targetNames, resolvedTarget.Name)
	}

	if len(components) > 0 {
		if len(normalizedTargets) > 0 {
			// When cross-compiling, components (notably stdx) belong to the
			// target SDK, so install them against each target's resolved name.
			for _, targetName := range targetNames {
				if err := installComponentsList(ctx, targetName, components, force, false); err != nil {
					return err
				}
			}
		} else if err := installComponentsList(ctx, resolved.Name, components, force, false); err != nil {
			return err
		}
	}
	return nil
}

// manifestFetcher fetches the SDK manifest at most once per install operation.
// The first call to get triggers the network fetch (and the FetchingManifest
// status line); subsequent calls return the cached result, including any error.
// The first caller's ctx is used for the actual fetch.
type manifestFetcher struct {
	once sync.Once
	url  string
	m    *dist.Manifest
	err  error
}

func newManifestFetcher(url string) *manifestFetcher {
	return &manifestFetcher{url: url}
}

func (f *manifestFetcher) get(ctx context.Context) (*dist.Manifest, error) {
	f.once.Do(func() {
		noteStep(i18n.T("FetchingManifest", nil))
		f.m, f.err = fetchManifest(ctx, f.url)
	})
	return f.m, f.err
}

// InstallComponentsForToolchain backs the proxy auto_install path: it
// resolves tcInput to an already-installed toolchain and installs missing
// components quietly.
func InstallComponentsForToolchain(ctx context.Context, tcInput string, components []string) error {
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
	return installComponentsList(ctx, filepath.Base(installedDir), components, false, true)
}

// installComponentsList expects resolvedName as "<channel>-<version>"
// (the directory name under <CJV_HOME>/toolchains/). quiet suppresses the
// per-component status lines; used by the proxy auto-install path.
func installComponentsList(ctx context.Context, resolvedName string, components []string, force, quiet bool) error {
	resolvedTC, err := toolchain.ParseToolchainName(resolvedName)
	if err != nil {
		return err
	}
	if resolvedTC.IsCustom() {
		return &cjverr.ComponentRequiresHostError{Component: strings.Join(components, ", ")}
	}
	parsed, err := componentlib.NormalizeList(components)
	if err != nil {
		return err
	}
	_, settings, err := clisettings.LoadSettings()
	if err != nil {
		return err
	}
	// For a target-variant resolved name (e.g. "lts-1.0.5-linux-x64-ohos") the
	// target tuple is encoded in the name; otherwise install against the host.
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
	roots, err := componentlib.RootsFor(resolvedName)
	if err != nil {
		return err
	}
	snap, err := componentlib.TakeSnapshot(roots, parsed)
	if err != nil {
		return err
	}
	defer snap.Cleanup() //nolint:errcheck // best-effort cleanup
	for _, c := range parsed {
		if err := componentInstallFunc(ctx, roots, resolvedTC, c, tuple, downloadsDir, force); err != nil {
			var alreadyErr *cjverr.ComponentAlreadyInstalledError
			if errors.As(err, &alreadyErr) {
				if !quiet && !output.IsJSON() {
					fmt.Println(err)
				}
				continue
			}
			_ = snap.Restore() //nolint:errcheck // best-effort rollback
			return err
		}
		if !quiet && !output.IsJSON() {
			color.Green(i18n.T("ComponentInstalled", i18n.MsgData{
				"Toolchain": resolvedName,
				"Component": string(c),
			}))
		}
	}
	return nil
}

// resolvedToolchain holds the result of toolchain resolution.
type resolvedToolchain struct {
	Name        string // e.g. "lts-1.0.5"
	URL         string // download URL
	SHA256      string // expected checksum (empty for nightly)
	ArchiveName string // display filename from the manifest when available
	Tuple       string // manifest platform tuple used to select the archive
}

func installResolved(ctx context.Context, rt resolvedToolchain, settings *config.Settings, sf *config.SettingsFile, force bool) (retErr error) {
	return installResolvedWithDefault(ctx, rt, settings, sf, force, true)
}

func installResolvedNoDefault(ctx context.Context, rt resolvedToolchain, settings *config.Settings, sf *config.SettingsFile, force bool) (retErr error) {
	return installResolvedWithDefault(ctx, rt, settings, sf, force, false)
}

func installResolvedWithDefault(ctx context.Context, rt resolvedToolchain, settings *config.Settings, sf *config.SettingsFile, force bool, allowDefault bool) (retErr error) {
	resolvedName := rt.Name
	tcDir, err := config.ToolchainsDir()
	if err != nil {
		return err
	}
	destDir := filepath.Join(tcDir, resolvedName)
	isReinstall := false
	if _, err := os.Stat(destDir); err == nil {
		if !force {
			if output.IsJSON() {
				// In JSON mode treat "already installed" as a structured error
				// so consumers can branch on it instead of seeing a no-op.
				return &cjverr.ToolchainAlreadyInstalledError{Name: resolvedName}
			}
			fmt.Println(i18n.T("ToolchainAlreadyInstalled", i18n.MsgData{
				"Name": resolvedName,
			}))
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
	// Drop the staged archive on success; failures keep it for the next retry.
	defer func() {
		if retErr == nil {
			_ = dist.CleanupDownload(archivePath) //nolint:errcheck // best-effort
		}
	}()

	stagingDir := destDir + toolchain.StagingSuffix
	if err := utils.RemoveAllRetry(stagingDir); err != nil {
		return fmt.Errorf("failed to clean staging directory: %w", err)
	}
	defer func() {
		if retErr != nil {
			_ = utils.RemoveAllRetry(stagingDir)
		}
	}()

	noteStep(i18n.T("Extracting", nil))
	if err := dist.InstallSDK(ctx, archivePath, stagingDir); err != nil {
		return err
	}

	if err := validateInstallation(stagingDir, rt.Tuple); err != nil {
		return err
	}

	isFirstInstall := allowDefault && (settings.DefaultToolchain == "" || !defaultToolchainExists(settings.DefaultToolchain))
	if err := swapInstalledToolchain(stagingDir, destDir, isReinstall, func() error {
		if _, err := selfupdate.EnsureManagedExecutable(); err != nil {
			return err
		}
		if err := proxy.CreateAllProxyLinks(); err != nil {
			return err
		}
		if isFirstInstall {
			settings.DefaultToolchain = resolvedName
			if err := sf.Save(settings); err != nil {
				return err
			}
			ensurePathConfiguredFn()
		}
		return nil
	}); err != nil {
		return err
	}

	if !output.IsJSON() {
		color.Green(i18n.T("ToolchainInstalled", i18n.MsgData{
			"Name": resolvedName,
		}))
	}
	return nil
}

// ensurePathConfigured adds the cjv bin directory to the user's PATH
// on first install, so proxy commands are immediately available.
//
// Set CJV_NO_PATH_SETUP=1 to skip PATH modification (useful for CI
// environments and integration tests).
func ensurePathConfigured() {
	if os.Getenv(config.EnvNoPathSetup) == "1" {
		return
	}

	binDir, err := config.BinDir()
	if err != nil {
		return
	}

	var pathErr error

	if runtime.GOOS == "windows" {
		if err := env.AddPathToWindowsRegistry(binDir); err != nil {
			slog.Warn("failed to add PATH to Windows registry", "error", err)
			pathErr = err
		}
	} else {
		posix, fish := env.ShellConfigPaths()
		for _, rc := range posix {
			if err := env.AddPathToShellConfig(rc, binDir); err != nil {
				slog.Warn("failed to add PATH to shell config", "file", rc, "error", err)
				pathErr = err
			}
		}
		if fish != "" {
			if err := env.AddPathToFishConfig(fish, binDir); err != nil {
				slog.Warn("failed to add PATH to fish config", "file", fish, "error", err)
				pathErr = err
			}
		}
	}

	if pathErr != nil {
		fmt.Fprintf(os.Stderr, "\n%s\n", i18n.T("PathConfigWarning", i18n.MsgData{"BinDir": binDir}))
	}
}

func resolveAndLocate(ctx context.Context, name toolchain.ToolchainName, settings *config.Settings, fetcher *manifestFetcher) (resolvedToolchain, error) {
	return resolveAndLocateWithTarget(ctx, name, settings, fetcher, "")
}

func resolveAndLocateWithTarget(ctx context.Context, name toolchain.ToolchainName, settings *config.Settings, fetcher *manifestFetcher, target string) (resolvedToolchain, error) {
	tuple := name.Target
	if tuple == "" {
		var err error
		tuple, err = dist.CurrentTargetTuple(settings.DefaultHost, target)
		if err != nil {
			return resolvedToolchain{}, err
		}
	}
	return resolveAndLocateWithTuple(ctx, name, settings, fetcher, tuple)
}

func resolveAndLocateWithTuple(ctx context.Context, name toolchain.ToolchainName, settings *config.Settings, fetcher *manifestFetcher, tuple string) (resolvedToolchain, error) {
	if tuple == "" {
		var err error
		tuple, err = dist.CurrentHostTuple(settings.DefaultHost)
		if err != nil {
			return resolvedToolchain{}, err
		}
	}
	if name.Channel == toolchain.Nightly {
		return resolveNightlyWithTuple(ctx, name, settings, tuple)
	}

	manifest, err := fetcher.get(ctx)
	if err != nil {
		return resolvedToolchain{}, err
	}

	channel := name.Channel
	version := name.Version

	// If channel is unknown (bare version number), find which channel it belongs to
	if channel == toolchain.UnknownChannel {
		found, err := manifest.FindVersionChannel(version)
		if err != nil {
			return resolvedToolchain{}, err
		}
		channel = found
	}

	if version == "" {
		v, err := latestVersionForTuple(manifest, channel, tuple)
		if err != nil {
			return resolvedToolchain{}, err
		}
		version = v
	}

	resolved := toolchain.ToolchainName{Channel: channel, Version: version}
	if parts, err := sdktarget.ParseTuple(tuple); err == nil && parts.Environment != "" {
		resolved.Target = tuple
	}

	info, err := manifest.GetDownloadInfo(channel, version, tuple)
	if err != nil {
		return resolvedToolchain{}, err
	}

	return resolvedToolchain{Name: resolved.String(), URL: info.URL, SHA256: info.SHA256, ArchiveName: info.Name, Tuple: tuple}, nil
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

func resolveNightlyWithTuple(ctx context.Context, name toolchain.ToolchainName, settings *config.Settings, tuple string) (resolvedToolchain, error) {
	if tuple == "" {
		var err error
		tuple, err = dist.CurrentHostTuple(settings.DefaultHost)
		if err != nil {
			return resolvedToolchain{}, err
		}
	}
	version := name.Version

	if version == "" {
		noteStep(i18n.T("FetchingNightly", nil))
		v, err := dist.FetchLatestNightly(ctx, dist.DefaultNightlyAPIURL, settings.GitCodeAPIKey)
		if err != nil {
			return resolvedToolchain{}, err
		}
		version = v
	}

	resolved := toolchain.ToolchainName{Channel: toolchain.Nightly, Version: version}
	if parts, err := sdktarget.ParseTuple(tuple); err == nil && parts.Environment != "" {
		resolved.Target = tuple
	}

	url, err := dist.NightlyDownloadURLForTuple(dist.DefaultNightlyBaseURL, version, tuple)
	if err != nil {
		return resolvedToolchain{}, err
	}

	sha256 := dist.FetchNightlySHA256(ctx, url)
	if sha256 == "" {
		noteStep(i18n.T("NightlyNoChecksum", nil))
	}
	return resolvedToolchain{Name: resolved.String(), URL: url, SHA256: sha256, Tuple: tuple}, nil
}

func fetchManifest(ctx context.Context, manifestURL string) (*dist.Manifest, error) {
	u, err := url.Parse(manifestURL)
	if err != nil {
		return nil, fmt.Errorf("invalid manifest URL: %w", err)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return nil, fmt.Errorf("invalid manifest URL scheme %q: only https and http are supported", u.Scheme)
	}
	if u.Scheme == "http" {
		slog.Warn("manifest URL uses insecure HTTP; consider using HTTPS", "url", manifestURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, manifestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create manifest request: %w", err)
	}

	resp, err := dist.HTTPClient().Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // best-effort cleanup

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to fetch manifest: HTTP %d", resp.StatusCode)
	}

	data, err := io.ReadAll(io.LimitReader(resp.Body, dist.MaxResponseSize))
	if err != nil {
		return nil, err
	}

	return dist.ParseManifest(data)
}

// defaultToolchainExists checks whether the configured default toolchain is still installed.
func defaultToolchainExists(name string) bool {
	parsed, err := toolchain.ParseToolchainName(name)
	if err != nil {
		return false
	}
	_, err = toolchain.FindInstalled(parsed)
	return err == nil
}

// validateInstallation checks that the installed SDK has essential binaries.
func validateInstallation(dir, tuple string) error {
	var err error
	if tuple == "" {
		_, err = proxy.ResolveInstalledToolBinary(dir, "cjc")
	} else {
		_, err = proxy.ResolveInstalledToolBinaryForTuple(dir, "cjc", tuple)
	}
	if err != nil {
		return fmt.Errorf("installation validation failed: %w", err)
	}
	return nil
}
