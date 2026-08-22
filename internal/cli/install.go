package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"github.com/Zxilly/cjv/internal/cli/output"
	"github.com/Zxilly/cjv/internal/cli/selfmgmt"
	componentlib "github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/env"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/Zxilly/cjv/internal/selfupdate"
	"github.com/Zxilly/cjv/internal/toolchain"
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
	// componentInstallFunc is a test seam: when nil (production) the lifecycle
	// default path installs components (resolving LTS / STS links from the
	// manifest); tests set it to stub the installer without touching the network.
	componentInstallFunc func(context.Context, componentlib.Roots, toolchain.ToolchainName, componentlib.Name, string, string, bool) error

	installToolchainWithExtrasFn = InstallToolchainWithExtras
)

func lifecycleOptions() lifecycle.Options {
	return lifecycle.Options{
		IsJSON:               output.IsJSON,
		EnsurePathConfigured: ensurePathConfiguredFn,
		ComponentInstall:     componentInstallFunc,
		EnsureManagedBinary:  selfupdate.EnsureManagedExecutable,
		CreateProxyLinks:     proxy.CreateAllProxyLinks,
		ValidateInstallation: validateInstallation,
	}
}

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
	return lifecycle.InstallToolchainWithExtras(ctx, input, targets, components, force, lifecycleOptions())
}

// manifestFetcher fetches the SDK manifest at most once per install operation.
// The first call to get triggers the network fetch (and the FetchingManifest
// status line); subsequent calls return the cached result, including any error.
// The first caller's ctx is used for the actual fetch.
type manifestFetcher struct {
	inner *lifecycle.ManifestFetcher
}

func newManifestFetcher(url string) *manifestFetcher {
	return &manifestFetcher{inner: lifecycle.NewManifestFetcher(url, lifecycleOptions())}
}

func newManifestFetcherForSettings(settings *config.Settings) (*manifestFetcher, error) {
	inner, err := lifecycle.NewManifestFetcherForSettings(settings, lifecycleOptions())
	if err != nil {
		return nil, err
	}
	return &manifestFetcher{inner: inner}, nil
}

func (f *manifestFetcher) get(ctx context.Context) (*dist.Manifest, error) {
	return f.inner.Get(ctx)
}

// InstallComponentsForToolchain backs the proxy auto_install path: it
// resolves tcInput to an already-installed toolchain and installs missing
// components quietly.
func InstallComponentsForToolchain(ctx context.Context, tcInput string, components []string) error {
	return lifecycle.InstallComponentsForToolchain(ctx, tcInput, components, lifecycleOptions())
}

// installComponentsList expects resolvedName as "<channel>-<version>"
// (the directory name under <CJV_HOME>/toolchains/). quiet suppresses the
// per-component status lines; used by the proxy auto-install path.
func installComponentsList(ctx context.Context, resolvedName string, components []string, force, quiet bool) error {
	return lifecycle.InstallComponentsList(ctx, resolvedName, components, force, quiet, nil, lifecycleOptions())
}

type resolvedToolchain = lifecycle.ResolvedToolchain

func installResolved(ctx context.Context, rt resolvedToolchain, settings *config.Settings, sf *config.SettingsFile, force bool) (retErr error) {
	return lifecycle.InstallResolved(ctx, rt, settings, sf, force, lifecycleOptions())
}

func installResolvedNoDefault(ctx context.Context, rt resolvedToolchain, settings *config.Settings, sf *config.SettingsFile, force bool) (retErr error) {
	return lifecycle.InstallResolvedNoDefault(ctx, rt, settings, sf, force, lifecycleOptions())
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

func resolveAndLocate(ctx context.Context, name toolchain.ToolchainName, settings *config.Settings, fetcher *manifestFetcher, tuple string) (resolvedToolchain, error) {
	return lifecycle.ResolveAndLocatePlatform(ctx, name, settings, fetcher.inner, tuple)
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
