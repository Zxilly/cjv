package cli

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Zxilly/cjv/internal/cli/selfmgmt"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/selfupdate"
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
	if len(args) == 0 && app.selector != "" {
		args = []string{app.selector}
	}
	if len(args) > 0 {
		var outcomes []lifecycle.UpdateOutcome
		var errs []error
		for _, input := range args {
			name, err := toolchain.ParseToolchainName(input)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			outcome, err := lifecycle.UpdateInstalled(ctx, name, app.lifecycleOptions())
			if err != nil {
				errs = append(errs, err)
				continue
			}
			outcomes = append(outcomes, outcome)
		}
		return app.output.RenderOutcome(cmdOutput(cmd), updateResult{Updates: updateEntries(outcomes)}, errors.Join(errs...))
	}

	report, err := lifecycle.UpdateAll(ctx, app.lifecycleOptions())
	result := updateResult{Updates: updateEntries(report.Outcomes), NoneInstalled: report.NoneInstalled}
	// Ordinary update errors can still leave partial success. Cancellation
	// suppresses self-update as well; it must not bootstrap a managed binary
	// after the requested operation has already stopped.
	if !report.NoneInstalled {
		app.autoSelfUpdate(ctx, &result)
	}
	return app.output.RenderOutcome(cmdOutput(cmd), result, err)
}

// autoSelfUpdate applies the auto_self_update setting after a full update and
// records the outcome on result: the applied update for the JSON payload and
// its text, or the current version after a check.
func (app *application) autoSelfUpdate(ctx context.Context, result *updateResult) {
	if ctx.Err() != nil {
		return
	}
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
		checked, checkErr := selfupdate.Check(ctx, app.updateURL, app.version)
		if checkErr != nil {
			slog.Warn("self-update check failed", "error", checkErr)
			return
		}
		result.SelfUpdate = &selfmgmt.UpdateResult{Version: checked.Version, Status: checked.Status}
		if checked.Status == selfupdate.StatusAvailable {
			result.selfUpdateText = fmt.Sprintf("  cjv %s → %s", app.version, checked.Version)
			result.checkedVersion = ""
		}
	}
}

func (app *application) initUpdateCommands() {
	app.updateCmd = &cobra.Command{
		Use:   "update [toolchain]...",
		Short: i18n.T("UpdateCmdShort", nil),
		Long:  i18n.T("UpdateCmdLong", nil),
		Args:  cobra.ArbitraryArgs,
		RunE:  app.runUpdate,
	}

	app.updateCmd.Flags().BoolVar(&app.noSelfUpdate, "no-self-update", false, i18n.T("UpdateFlagNoSelfUpdate", nil))
	app.updateCmd.Flags().BoolVar(&app.forceUpdate, "force", false, "Update even when optional components or targets are unavailable")
	app.updateCmd.Flags().BoolVar(&app.allowDowngrade, "allow-downgrade", false, "Allow an older compatible nightly")
	app.rootCmd.AddCommand(app.updateCmd)

}
