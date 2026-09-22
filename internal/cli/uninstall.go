package cli

import (
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/charmbracelet/huh"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

func (app *application) runUninstall(cmd *cobra.Command, args []string) error {
	name := args[0]

	if err := lifecycle.PrepareToolchainRemoval(name); err != nil {
		return err
	}

	// Confirm before destroying the toolchain and its components. Skipped with
	// --yes, in JSON mode, or when stdin is not an interactive terminal (so
	// scripts/CI are not blocked waiting on a prompt).
	if !app.uninstallYes && !app.output.IsJSON() && initStdinIsTerminal() {
		confirm := false
		if err := huh.NewConfirm().
			Title(i18n.T("ToolchainUninstallConfirm", i18n.MsgData{"Name": name})).
			Value(&confirm).
			Run(); err != nil {
			return err
		}
		if !confirm {
			return nil
		}
	}

	if err := lifecycle.RemoveToolchain(name); err != nil {
		return err
	}

	return app.output.RenderTo(cmdOutput(cmd), uninstallResult{Name: name})
}

type uninstallResult struct {
	Name string `json:"name"`
}

func (r uninstallResult) Text() string {
	return color.GreenString(i18n.T("ToolchainUninstalled", i18n.MsgData{"Name": r.Name}))
}

func (app *application) initUninstallCommands() {
	app.uninstallCmd = &cobra.Command{
		Use:   "uninstall <toolchain>",
		Short: i18n.T("UninstallCmdShort", nil),
		Args:  cobra.ExactArgs(1),
		RunE:  app.runUninstall,
	}

	app.uninstallCmd.Flags().BoolVarP(&app.uninstallYes, "yes", "y", false, i18n.T("FlagSkipConfirm", nil))
	app.rootCmd.AddCommand(app.uninstallCmd)

}
