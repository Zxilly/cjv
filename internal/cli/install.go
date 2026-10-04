package cli

import (
	"errors"
	"github.com/Zxilly/cjv/internal/cli/output"
	"github.com/Zxilly/cjv/internal/cli/selfmgmt"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/progress"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
)

// progress is the adapter this invocation's operations report to: the
// renderer picks text on the command writer or nothing in JSON mode.
func (app *application) progress() progress.Sink {
	return app.output.Progress(app.rootCmd.OutOrStdout())
}

func (app *application) lifecycleOptions() lifecycle.Options {
	return lifecycle.Options{
		Progress:       app.progress(),
		ConfigurePath:  true,
		AllowMissing:   app.forceUpdate || app.forceInstall,
		AllowDowngrade: app.allowDowngrade,
	}
}

type installResult struct {
	output.ProgressDriven
	Toolchain  string   `json:"toolchain"`
	Targets    []string `json:"targets"`
	Components []string `json:"components"`
	Forced     bool     `json:"forced"`
}

type installBatchResult struct {
	output.ProgressDriven
	Installations []installResult `json:"installations"`
}

func (app *application) runInstall(cmd *cobra.Command, args []string) error {
	selfmgmt.CheckSudoSafety()
	if len(args) == 0 {
		name, err := app.selectedToolchain("")
		if err != nil {
			return err
		}
		args = []string{name}
	}
	var errs []error
	var results []installResult
	for _, input := range args {
		err := lifecycle.Install(cmd.Context(), lifecycle.InstallRequest{
			Toolchain:  input,
			Targets:    app.installTargets,
			Components: app.installComponents,
			Force:      app.forceInstall,
			NoUpdate:   app.noUpdate,
		}, app.lifecycleOptions())
		if err != nil {
			errs = append(errs, err)
		} else {
			results = append(results, installResult{Toolchain: input, Targets: app.installTargets, Components: app.installComponents, Forced: app.forceInstall})
		}
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}
	if len(args) > 1 {
		return app.output.RenderTo(cmdOutput(cmd), installBatchResult{Installations: results})
	}
	return app.output.RenderTo(cmdOutput(cmd), installResult{
		Toolchain:  args[0],
		Targets:    app.installTargets,
		Components: app.installComponents,
		Forced:     app.forceInstall,
	})
}

func (app *application) initInstallCommands() {
	app.installCmd = &cobra.Command{
		Use:   "install [toolchain]...",
		Short: i18n.T("InstallCmdShort", nil),
		Args:  cobra.ArbitraryArgs,
		RunE:  app.runInstall,
	}

	app.installCmd.Flags().BoolVar(&app.forceInstall, "force", false, i18n.T("InstallFlagForce", nil))
	app.installCmd.Flags().StringSliceVarP(&app.installTargets, "target", "t", nil, i18n.T("InstallFlagTarget", nil))
	app.installCmd.Flags().StringSliceVarP(&app.installComponents, "component", "c", nil, i18n.T("InstallFlagComponent", nil))
	app.installCmd.Flags().BoolVar(&app.noUpdate, "no-update", false, "Keep an existing SDK release while adding targets or components")
	app.installCmd.Flags().BoolVar(&app.allowDowngrade, "allow-downgrade", false, "Allow an older nightly that supplies the requested components")
	app.installCmd.MarkFlagsMutuallyExclusive("force", "no-update")
	app.rootCmd.AddCommand(app.installCmd)

}

func (app *application) selectedToolchain(flag string) (string, error) {
	if err := toolchain.RecoverHome(); err != nil {
		return "", err
	}
	if flag != "" {
		return flag, nil
	}
	if app.selector != "" {
		return app.selector, nil
	}
	_, name, _, err := toolchain.ResolveActiveToolchain()
	// A configured missing toolchain is still a valid install selection.
	if name != "" {
		return name, nil
	}
	return "", err
}
