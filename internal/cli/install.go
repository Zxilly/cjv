package cli

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"

	"github.com/Zxilly/cjv/internal/cli/selfmgmt"
	clisettings "github.com/Zxilly/cjv/internal/cli/settings"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/env"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/Zxilly/cjv/internal/selfupdate"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/Zxilly/cjv/internal/utils"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	forceInstall bool

	// ensurePathConfiguredFn is called during first install to add cjv's bin
	// directory to the user's PATH. Tests override this to avoid writing to
	// the real system PATH (e.g., the Windows registry).
	ensurePathConfiguredFn = ensurePathConfigured
)

func init() {
	installCmd.Flags().BoolVar(&forceInstall, "force", false, "Force re-download and re-install even if already installed")
	rootCmd.AddCommand(installCmd)
}

var installCmd = &cobra.Command{
	Use:   "install <toolchain>",
	Short: "Install a Cangjie SDK toolchain",
	Args:  cobra.ExactArgs(1),
	RunE:  runInstall,
}

func runInstall(cmd *cobra.Command, args []string) error {
	selfmgmt.CheckSudoSafety()
	toolchain.CleanupStagingDirs()
	return InstallToolchainWithOptions(cmd.Context(), args[0], forceInstall)
}

// InstallToolchainWithOptions installs a toolchain with optional force re-install.
func InstallToolchainWithOptions(ctx context.Context, input string, force bool) error {
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

	resolved, err := resolveAndLocate(ctx, name, settings, nil)
	if err != nil {
		return err
	}

	return installResolved(ctx, resolved, settings, sf, force)
}

// resolvedToolchain holds the result of toolchain resolution.
type resolvedToolchain struct {
	Name   string // e.g. "lts-1.0.5"
	URL    string // download URL
	SHA256 string // expected checksum (empty for nightly)
}

func installResolved(ctx context.Context, rt resolvedToolchain, settings *config.Settings, sf *config.SettingsFile, force bool) (retErr error) {
	resolvedName := rt.Name
	tcDir, err := config.ToolchainsDir()
	if err != nil {
		return err
	}
	destDir := filepath.Join(tcDir, resolvedName)
	isReinstall := false
	if _, err := os.Stat(destDir); err == nil {
		if !force {
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
	u, err := url.Parse(rt.URL)
	if err != nil || u.Path == "" {
		return fmt.Errorf("invalid toolchain download URL: %s", rt.URL)
	}
	archivePath := filepath.Join(downloadsDir, filepath.Base(u.Path))

	if err := dist.DownloadFileCached(ctx, rt.URL, archivePath, rt.SHA256, downloadsDir); err != nil {
		return err
	}

	stagingDir := destDir + toolchain.StagingSuffix
	if err := utils.RemoveAllRetry(stagingDir); err != nil {
		return fmt.Errorf("failed to clean staging directory: %w", err)
	}
	defer func() {
		if retErr != nil {
			_ = utils.RemoveAllRetry(stagingDir)
		}
	}()

	fmt.Println(i18n.T("Extracting", nil))
	if err := dist.InstallSDK(ctx, archivePath, stagingDir); err != nil {
		return err
	}

	if err := validateInstallation(stagingDir); err != nil {
		return err
	}

	isFirstInstall := settings.DefaultToolchain == ""
	if err := swapInstalledToolchain(stagingDir, destDir, isReinstall, func() error {
		// Capture env after the rename so that $PWD = destDir and all
		// paths in env.toml naturally point to the final location.
		fmt.Println(i18n.T("CapturingEnv", nil))
		envCfg, err := env.CaptureEnvSetup(ctx, destDir)
		if err != nil {
			return err
		}
		if err := envCfg.Save(filepath.Join(destDir, "env.toml")); err != nil {
			return err
		}

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

	color.Green(i18n.T("ToolchainInstalled", i18n.MsgData{
		"Name": resolvedName,
	}))
	return nil
}

// ensurePathConfigured adds the cjv bin directory to the user's PATH
// on first install, so proxy commands are immediately available.
//
// Set CJV_NO_PATH_SETUP=1 to skip PATH modification (useful for CI
// environments and integration tests).
func ensurePathConfigured() {
	if os.Getenv("CJV_NO_PATH_SETUP") == "1" {
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

func resolveAndLocate(ctx context.Context, name toolchain.ToolchainName, settings *config.Settings, manifest *dist.Manifest) (resolvedToolchain, error) {
	if name.Channel == toolchain.Nightly {
		return resolveNightly(ctx, name)
	}

	platformKey, err := dist.CurrentPlatformKey(settings.DefaultHost)
	if err != nil {
		return resolvedToolchain{}, err
	}

	if manifest == nil {
		fmt.Println(i18n.T("FetchingManifest", nil))
		var fetchErr error
		manifest, fetchErr = fetchManifest(ctx, settings.ManifestURL)
		if fetchErr != nil {
			return resolvedToolchain{}, fetchErr
		}
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
		v, err := manifest.GetLatestVersion(channel)
		if err != nil {
			return resolvedToolchain{}, err
		}
		version = v
	}

	resolved := toolchain.ToolchainName{Channel: channel, Version: version}

	info, err := manifest.GetDownloadInfo(channel, version, platformKey)
	if err != nil {
		return resolvedToolchain{}, err
	}

	return resolvedToolchain{Name: resolved.String(), URL: info.URL, SHA256: info.SHA256}, nil
}

func resolveNightly(ctx context.Context, name toolchain.ToolchainName) (resolvedToolchain, error) {
	version := name.Version

	if version == "" {
		fmt.Println(i18n.T("FetchingNightly", nil))
		v, err := dist.FetchLatestNightly(ctx, dist.DefaultNightlyAPIURL)
		if err != nil {
			return resolvedToolchain{}, err
		}
		version = v
	}

	resolved := toolchain.ToolchainName{Channel: toolchain.Nightly, Version: version}

	url, err := dist.NightlyDownloadURL(dist.DefaultNightlyBaseURL, version, runtime.GOOS, runtime.GOARCH)
	if err != nil {
		return resolvedToolchain{}, err
	}

	fmt.Println(i18n.T("NightlyNoChecksum", nil))
	return resolvedToolchain{Name: resolved.String(), URL: url}, nil
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

// validateInstallation checks that the installed SDK has essential binaries.
func validateInstallation(dir string) error {
	if _, err := proxy.ResolveInstalledToolBinary(dir, "cjc"); err != nil {
		return fmt.Errorf("installation validation failed: %w", err)
	}
	return nil
}
