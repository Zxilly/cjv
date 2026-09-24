package cli

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Zxilly/cjv/internal/cli/selfmgmt"
	clisettings "github.com/Zxilly/cjv/internal/cli/settings"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
)

type updateEntry struct {
	From string `json:"from"`
	To   string `json:"to"`
}

type updateResult struct {
	Updates       []updateEntry          `json:"updates"`
	NoneInstalled bool                   `json:"none_installed,omitempty"`
	SelfUpdate    *selfmgmt.UpdateResult `json:"self_update,omitempty"`
}

func (r updateResult) Text() string { return "" }

func updateEntries(outcomes []lifecycle.UpdateOutcome) []updateEntry {
	var entries []updateEntry
	for _, o := range outcomes {
		if o.Status == lifecycle.UpdateApplied {
			entries = append(entries, updateEntry{From: o.Name, To: o.Replacement})
		}
	}
	return entries
}

func (app *application) runUpdate(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if err := toolchain.RecoverHome(); err != nil {
		slog.Warn("failed to recover install transaction", "error", err)
	}

	if len(args) == 1 {
		name, err := toolchain.ParseToolchainName(args[0])
		if err != nil {
			return err
		}
		outcome, err := lifecycle.UpdateInstalled(ctx, name, app.lifecycleOptions())
		if err != nil {
			return err
		}
		if !app.output.IsJSON() {
			return nil
		}
		return app.output.RenderTo(cmdOutput(cmd), updateResult{Updates: updateEntries([]lifecycle.UpdateOutcome{outcome})})
	}

	report, err := lifecycle.UpdateAll(ctx, app.lifecycleOptions())
	if report.NoneInstalled && !app.output.IsJSON() {
		fmt.Println(i18n.T("NoToolchainsInstalled", nil))
	}

	// The self-update decision runs regardless of UpdateAll errors: it may
	// have partially succeeded, and the original error is returned at the end.
	var selfUpdate *selfmgmt.UpdateResult
	if !report.NoneInstalled {
		var renderErr error
		selfUpdate, renderErr = app.autoSelfUpdate(ctx, cmd)
		if renderErr != nil {
			return renderErr
		}
	}

	if err != nil {
		return err
	}
	if !app.output.IsJSON() {
		return nil
	}
	return app.output.RenderTo(cmdOutput(cmd), updateResult{
		Updates:       updateEntries(report.Outcomes),
		NoneInstalled: report.NoneInstalled,
		SelfUpdate:    selfUpdate,
	})
}

// autoSelfUpdate applies the auto_self_update setting after a full update.
// The result is returned for the JSON envelope; in text mode an applied
// self-update is rendered here and a check prints the current version.
func (app *application) autoSelfUpdate(ctx context.Context, cmd *cobra.Command) (*selfmgmt.UpdateResult, error) {
	_, settings, err := clisettings.LoadSettings()
	if err != nil || app.noSelfUpdate || settings.AutoSelfUpdate == config.AutoSelfUpdateDisable {
		return nil, nil
	}
	switch settings.AutoSelfUpdate {
	case config.AutoSelfUpdateEnable:
		var selfUpdate *selfmgmt.UpdateResult
		selfResult, selfErr := selfmgmt.UpdateManaged(ctx, app.updateURL, app.version)
		if selfResult.Status != "" {
			selfUpdate = &selfResult
		}
		if selfErr != nil {
			slog.Warn("self-update failed", "error", selfErr)
		} else if !app.output.IsJSON() {
			if renderErr := app.output.RenderTo(cmdOutput(cmd), selfResult); renderErr != nil {
				return selfUpdate, renderErr
			}
		}
		return selfUpdate, nil
	default:
		if settings.AutoSelfUpdate != config.AutoSelfUpdateCheck {
			slog.Warn("unknown auto_self_update value, treating as check", "value", settings.AutoSelfUpdate)
		}
		if !app.output.IsJSON() {
			fmt.Printf("\n  cjv %s\n", app.version)
		}
		return nil, nil
	}
}

func (app *application) initUpdateCommands() {
	app.updateCmd = &cobra.Command{
		Use:   "update [toolchain]",
		Short: i18n.T("UpdateCmdShort", nil),
		Long:  i18n.T("UpdateCmdLong", nil),
		Args:  cobra.MaximumNArgs(1),
		RunE:  app.runUpdate,
	}

	app.updateCmd.Flags().BoolVar(&app.noSelfUpdate, "no-self-update", false, i18n.T("UpdateFlagNoSelfUpdate", nil))
	app.rootCmd.AddCommand(app.updateCmd)

}
