package cli

import (
	"github.com/Zxilly/cjv/internal/cli/output"
	"github.com/Zxilly/cjv/internal/cli/selfmgmt"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/progress"
	"github.com/spf13/cobra"
)

// progress is the adapter this invocation's operations report to: the
// renderer picks text on the command writer or nothing in JSON mode.
func (app *application) progress() progress.Sink {
	return app.output.Progress(app.rootCmd.OutOrStdout())
}

func (app *application) lifecycleOptions() lifecycle.Options {
	return lifecycle.Options{
		Progress:      app.progress(),
		ConfigurePath: true,
	}
}

type installResult struct {
	output.ProgressDriven
	Toolchain  string   `json:"toolchain"`
	Targets    []string `json:"targets"`
	Components []string `json:"components"`
	Forced     bool     `json:"forced"`
}

func (app *application) runInstall(cmd *cobra.Command, args []string) error {
	selfmgmt.CheckSudoSafety()
	err := lifecycle.Install(cmd.Context(), lifecycle.InstallRequest{
		Toolchain:  args[0],
		Targets:    app.installTargets,
		Components: app.installComponents,
		Force:      app.forceInstall,
	}, app.lifecycleOptions())
	if err != nil {
		return err
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
