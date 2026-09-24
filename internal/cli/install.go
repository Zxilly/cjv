package cli

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"runtime"

	"github.com/Zxilly/cjv/internal/cli/selfmgmt"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/env"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/Zxilly/cjv/internal/selfupdate"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// reportProgress owns presentation for installation and component operations.
func (app *application) reportProgress(message string, data i18n.MsgData) {
	if app.output.IsJSON() {
		return
	}
	text := i18n.T(message, data)
	if message == "ToolchainInstalled" || message == "ComponentInstalled" {
		text = color.GreenString("%s", text)
	}
	_, _ = fmt.Fprintln(app.rootCmd.OutOrStdout(), text)
}

func (app *application) lifecycleOptions() lifecycle.Options {
	return lifecycle.Options{
		Report:               app.reportProgress,
		EnsurePathConfigured: app.ensurePathConfiguredFn,
		ComponentInstall:     app.componentInstallFunc,
		EnsureManagedBinary:  selfupdate.EnsureManagedExecutable,
		CreateProxyLinks:     proxy.CreateAllProxyLinks,
		ValidateInstallation: validateInstallation,
	}
}

type installResult struct {
	Toolchain  string   `json:"toolchain"`
	Targets    []string `json:"targets"`
	Components []string `json:"components"`
	Forced     bool     `json:"forced"`
}

func (r installResult) Text() string { return "" }

func (app *application) runInstall(cmd *cobra.Command, args []string) error {
	selfmgmt.CheckSudoSafety()
	if err := app.InstallToolchainWithExtras(cmd.Context(), args[0], app.installTargets, app.installComponents, app.forceInstall); err != nil {
		return err
	}
	if !app.output.IsJSON() {
		return nil
	}
	return app.output.RenderTo(cmdOutput(cmd), installResult{
		Toolchain:  args[0],
		Targets:    app.installTargets,
		Components: app.installComponents,
		Forced:     app.forceInstall,
	})
}

// InstallToolchainWithOptions installs a toolchain with optional force re-install.
func (app *application) InstallToolchainWithOptions(ctx context.Context, input string, force bool) error {
	return app.InstallToolchainWithExtras(ctx, input, nil, nil, force)
}

// InstallToolchainWithTargets installs the host toolchain plus optional cross SDK target variants.
func (app *application) InstallToolchainWithTargets(ctx context.Context, input string, targets []string, force bool) error {
	return app.InstallToolchainWithExtras(ctx, input, targets, nil, force)
}

// InstallToolchainWithExtras installs the host toolchain plus optional cross
// SDK target variants and optional components.
func (app *application) InstallToolchainWithExtras(ctx context.Context, input string, targets, components []string, force bool) error {
	return lifecycle.InstallToolchainWithExtras(ctx, input, targets, components, force, app.lifecycleOptions())
}

func (app *application) newManifestFetcher(url string) *lifecycle.ManifestFetcher {
	return lifecycle.NewManifestFetcher(url, app.lifecycleOptions())
}

func (app *application) newManifestFetcherForSettings(settings *config.Settings) (*lifecycle.ManifestFetcher, error) {
	return lifecycle.NewManifestFetcherForSettings(settings, app.lifecycleOptions())
}

// InstallComponentsForToolchain backs the proxy auto_install path: it
// resolves tcInput to an already-installed toolchain and installs missing
// components quietly.
func (app *application) InstallComponentsForToolchain(ctx context.Context, tcInput string, components []string) error {
	return lifecycle.InstallComponentsForToolchain(ctx, tcInput, components, app.lifecycleOptions())
}

// installComponentsList expects resolvedName as "<channel>-<version>"
// (the directory name under <CJV_HOME>/toolchains/). quiet suppresses the
// per-component status lines; used by the proxy auto-install path.
func (app *application) installComponentsList(ctx context.Context, resolvedName string, components []string, force, quiet bool) error {
	return lifecycle.InstallComponentsList(ctx, resolvedName, components, force, quiet, nil, app.lifecycleOptions())
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

func resolveAndLocate(ctx context.Context, name toolchain.ToolchainName, settings *config.Settings, fetcher *lifecycle.ManifestFetcher, tuple string) (lifecycle.ResolvedToolchain, error) {
	return lifecycle.ResolveAndLocatePlatform(ctx, name, settings, fetcher, tuple)
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

func (app *application) initInstallCommands() {
	app.ensurePathConfiguredFn = ensurePathConfigured
	app.installToolchainWithExtrasFn = app.InstallToolchainWithExtras
	app.installCmd = &cobra.Command{
		Use:   "install <toolchain>",
		Short: i18n.T("InstallCmdShort", nil),
		Args:  cobra.ExactArgs(1),
		RunE:  app.runInstall,
	}

	app.installCmd.Flags().BoolVar(&app.forceInstall, "force", false, i18n.T("InstallFlagForce", nil))
	app.installCmd.Flags().StringSliceVarP(&app.installTargets, "target", "t", nil, i18n.T("InstallFlagTarget", nil))
	app.installCmd.Flags().StringSliceVarP(&app.installComponents, "component", "c", nil, i18n.T("InstallFlagComponent", nil))
	app.rootCmd.AddCommand(app.installCmd)

}
