package cli

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Zxilly/cjv/internal/cli/selfmgmt"
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

	// Text mode: the toolchain updates were already reported as progress;
	// what remains is the empty-home hint and the self-update outcome, either
	// the applied update's own text or the current version after a check.
	selfUpdateText string
	checkedVersion string
}

func (r updateResult) Text() string {
	if r.NoneInstalled {
		return i18n.T("NoToolchainsInstalled", nil)
	}
	var b strings.Builder
	if r.selfUpdateText != "" {
		b.WriteString(r.selfUpdateText)
		if !strings.HasSuffix(r.selfUpdateText, "\n") {
			b.WriteByte('\n')
		}
	}
	if r.checkedVersion != "" {
		fmt.Fprintf(&b, "\n  cjv %s\n", r.checkedVersion)
	}
	return b.String()
}

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
		return app.output.RenderTo(cmdOutput(cmd), updateResult{Updates: updateEntries([]lifecycle.UpdateOutcome{outcome})})
	}

	report, err := lifecycle.UpdateAll(ctx, app.lifecycleOptions())
	result := updateResult{Updates: updateEntries(report.Outcomes), NoneInstalled: report.NoneInstalled}
	// The self-update decision runs regardless of UpdateAll errors: it may
	// have partially succeeded, and the original error is returned at the end.
	if !report.NoneInstalled {
		app.autoSelfUpdate(ctx, &result)
	}
	return app.output.RenderOutcome(cmdOutput(cmd), result, err)
}

// autoSelfUpdate applies the auto_self_update setting after a full update and
// records the outcome on result: the applied update for the JSON payload and
// its text, or the current version after a check.
func (app *application) autoSelfUpdate(ctx context.Context, result *updateResult) {
	_, settings, err := config.LoadDefaultSettings()
	if err != nil || app.noSelfUpdate || settings.AutoSelfUpdate == config.AutoSelfUpdateDisable {
		return
	}
	switch settings.AutoSelfUpdate {
	case config.AutoSelfUpdateEnable:
		selfResult, selfErr := selfmgmt.UpdateManaged(ctx, app.updateURL, app.version, app.progress())
		if selfResult.Status != "" {
			result.SelfUpdate = &selfResult
		}
		if selfErr != nil {
			slog.Warn("self-update failed", "error", selfErr)
			return
		}
		result.selfUpdateText = selfResult.Text()
	default:
		if settings.AutoSelfUpdate != config.AutoSelfUpdateCheck {
			slog.Warn("unknown auto_self_update value, treating as check", "value", settings.AutoSelfUpdate)
		}
		result.checkedVersion = app.version
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
