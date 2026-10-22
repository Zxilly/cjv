package toolchaincmd

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/cli/selfcmd"
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
	Updates       []updateEntry         `json:"updates"`
	NoneInstalled bool                  `json:"none_installed,omitempty"`
	SelfUpdate    *selfcmd.UpdateResult `json:"self_update,omitempty"`

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

// update holds the flags of `cjv update` for one invocation.
type update struct {
	ctx            *clictx.Context
	force          bool
	allowDowngrade bool
	noSelfUpdate   bool
	dryRun         bool
}

func newUpdate(ctx *clictx.Context) *update {
	return &update{ctx: ctx}
}

func (c *update) lifecycleOptions() lifecycle.Options {
	opts := c.ctx.LifecycleOptions()
	opts.AllowMissing = c.force
	opts.AllowDowngrade = c.allowDowngrade
	return opts
}

func (c *update) run(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if len(args) == 0 && c.ctx.Selector != "" {
		args = []string{c.ctx.Selector}
	}
	if c.dryRun {
		plan, err := lifecycle.PlanUpdates(ctx, args, c.lifecycleOptions())
		return c.ctx.Output.RenderOutcome(clictx.Out(cmd), updatePlanResult{UpdatePlanReport: plan, DryRun: true}, err)
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
			outcome, err := lifecycle.UpdateInstalled(ctx, name, c.lifecycleOptions())
			if err != nil {
				errs = append(errs, err)
				continue
			}
			outcomes = append(outcomes, outcome)
		}
		return c.ctx.Output.RenderOutcome(clictx.Out(cmd), updateResult{Updates: updateEntries(outcomes)}, errors.Join(errs...))
	}

	report, err := lifecycle.UpdateAll(ctx, c.lifecycleOptions())
	result := updateResult{Updates: updateEntries(report.Outcomes), NoneInstalled: report.NoneInstalled}
	// Ordinary update errors can still leave partial success. Cancellation
	// suppresses self-update as well; it must not bootstrap a managed binary
	// after the requested operation has already stopped.
	if !report.NoneInstalled {
		c.autoSelfUpdate(ctx, &result)
	}
	return c.ctx.Output.RenderOutcome(clictx.Out(cmd), result, err)
}

// autoSelfUpdate applies the auto_self_update setting after a full update and
// records the outcome on result: the applied update for the JSON payload and
// its text, or the current version after a check.
func (c *update) autoSelfUpdate(ctx context.Context, result *updateResult) {
	if ctx.Err() != nil {
		return
	}
	_, settings, err := config.LoadDefaultSettings()
	if err != nil || c.noSelfUpdate || settings.AutoSelfUpdate == config.AutoSelfUpdateDisable {
		return
	}
	switch settings.AutoSelfUpdate {
	case config.AutoSelfUpdateEnable:
		selfResult, selfErr := selfcmd.UpdateManaged(ctx, c.ctx.UpdateURL, c.ctx.Version, c.ctx.Progress())
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
		result.checkedVersion = c.ctx.Version
		checked, checkErr := selfupdate.Check(ctx, c.ctx.UpdateURL, c.ctx.Version)
		if checkErr != nil {
			slog.Warn("self-update check failed", "error", checkErr)
			return
		}
		result.SelfUpdate = &selfcmd.UpdateResult{Version: checked.Version, Status: checked.Status}
		if checked.Status == selfupdate.StatusAvailable {
			result.selfUpdateText = fmt.Sprintf("  cjv %s → %s", c.ctx.Version, checked.Version)
			result.checkedVersion = ""
		}
	}
}

func (c *update) command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update [toolchain]...",
		Short: i18n.T("UpdateCmdShort", nil),
		Long:  i18n.T("UpdateCmdLong", nil),
		Args:  cobra.ArbitraryArgs,
		RunE:  c.run,
	}
	cmd.Flags().BoolVar(&c.noSelfUpdate, "no-self-update", false, i18n.T("UpdateFlagNoSelfUpdate", nil))
	cmd.Flags().BoolVar(&c.dryRun, "dry-run", false, "Show the compatible update plan without changing installed files")
	cmd.Flags().BoolVar(&c.force, "force", false, "Update even when optional components or targets are unavailable")
	cmd.Flags().BoolVar(&c.allowDowngrade, "allow-downgrade", false, "Allow an older compatible nightly")
	return cmd
}

type updatePlanResult struct {
	lifecycle.UpdatePlanReport
	DryRun bool `json:"dry_run"`
}

func (r updatePlanResult) Text() string {
	if r.NoneInstalled {
		return i18n.T("NoToolchainsInstalled", nil)
	}
	var b strings.Builder
	for _, plan := range r.Toolchains {
		fmt.Fprintf(&b, "%s: %s", plan.Name, plan.Status)
		if plan.Selected != "" {
			fmt.Fprintf(&b, " (%s", plan.Selected)
			if plan.Latest != "" && plan.Latest != plan.Selected {
				fmt.Fprintf(&b, "; latest %s", plan.Latest)
			}
			b.WriteString(")")
		}
		b.WriteByte('\n')
		for _, blocker := range plan.Blockers {
			fmt.Fprintf(&b, "  %s\n", blocker)
		}
		if plan.Error != "" {
			fmt.Fprintf(&b, "  %s\n", plan.Error)
		}
	}
	return b.String()
}
