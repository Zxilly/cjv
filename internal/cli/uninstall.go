package cli

import (
	"errors"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/charmbracelet/huh"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"strings"
)

func (app *application) runUninstall(cmd *cobra.Command, args []string) error {
	var errs []error
	var removed []uninstallResult
	for _, name := range args {
		result, err := app.runUninstallOne(cmd, name)
		if err != nil {
			errs = append(errs, err)
		} else if result != nil {
			removed = append(removed, *result)
		}
	}
	if len(args) == 1 && len(removed) == 1 {
		return app.output.RenderOutcome(cmdOutput(cmd), removed[0], errors.Join(errs...))
	}
	return app.output.RenderOutcome(cmdOutput(cmd), uninstallBatchResult{Removed: removed}, errors.Join(errs...))
}

func (app *application) runUninstallOne(cmd *cobra.Command, name string) (*uninstallResult, error) {

	if err := lifecycle.PrepareToolchainRemoval(name); err != nil {
		return nil, err
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
			return nil, err
		}
		if !confirm {
			return nil, nil
		}
	}

	if err := lifecycle.RemoveToolchain(name); err != nil {
		return nil, err
	}

	return &uninstallResult{Name: name}, nil
}

type uninstallBatchResult struct {
	Removed []uninstallResult `json:"removed"`
}

func (r uninstallBatchResult) Text() string {
	var b strings.Builder
	for _, result := range r.Removed {
		b.WriteString(result.Text())
		b.WriteByte('\n')
	}
	return b.String()
}

type uninstallResult struct {
	Name string `json:"name"`
}

func (r uninstallResult) Text() string {
	return color.GreenString(i18n.T("ToolchainUninstalled", i18n.MsgData{"Name": r.Name}))
}

func (app *application) initUninstallCommands() {
	app.uninstallCmd = &cobra.Command{
		Use:   "uninstall <toolchain>...",
		Short: i18n.T("UninstallCmdShort", nil),
		Args:  cobra.MinimumNArgs(1),
		RunE:  app.runUninstall,
	}

	app.uninstallCmd.Flags().BoolVarP(&app.uninstallYes, "yes", "y", false, i18n.T("FlagSkipConfirm", nil))
	app.rootCmd.AddCommand(app.uninstallCmd)

}
