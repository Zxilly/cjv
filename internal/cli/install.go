package cli

import (
	"fmt"

	"github.com/Zxilly/cjv/internal/cli/selfmgmt"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/lifecycle"
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
		Report:           app.reportProgress,
		ConfigurePath:    true,
		ComponentInstall: app.componentInstallFunc,
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
	err := lifecycle.Install(cmd.Context(), lifecycle.InstallRequest{
		Toolchain:  args[0],
		Targets:    app.installTargets,
		Components: app.installComponents,
		Force:      app.forceInstall,
	}, app.lifecycleOptions())
	if err != nil {
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
