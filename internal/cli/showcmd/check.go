package showcmd

import (
	"fmt"
	"slices"
	"strings"

	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/selfupdate"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

type checkEntry struct {
	Name            string   `json:"name"`
	Latest          string   `json:"latest,omitempty"`
	UpdateAvailable bool     `json:"update_available"`
	NotForTarget    bool     `json:"not_for_target,omitempty"`
	Target          string   `json:"target,omitempty"`
	Error           string   `json:"error,omitempty"`
	Current         string   `json:"current,omitempty"`
	Selected        string   `json:"selected,omitempty"`
	Status          string   `json:"status,omitempty"`
	Blockers        []string `json:"blockers,omitempty"`
}

type checkResult struct {
	LatestCjvVersion string       `json:"latest_cjv_version,omitempty"`
	SelfUpdateError  string       `json:"self_update_error,omitempty"`
	Toolchains       []checkEntry `json:"toolchains"`
	CjvVersion       string       `json:"cjv_version"`
	HasUpdates       bool         `json:"has_updates"`
	NoneInstalled    bool         `json:"none_installed,omitempty"`
}

func (r checkResult) Text() string {
	if r.NoneInstalled && r.LatestCjvVersion == "" && r.SelfUpdateError == "" {
		return i18n.T("NoToolchainsInstalled", nil)
	}
	var b strings.Builder
	for _, e := range r.Toolchains {
		switch {
		case e.Error != "":
			fmt.Fprintf(&b, "  %s: %s\n", e.Name, e.Error)
		case len(e.Blockers) > 0:
			fmt.Fprintf(&b, i18n.T("CheckSelectedVersion", nil), e.Name, e.Selected, e.Latest)
			for _, blocker := range e.Blockers {
				fmt.Fprintf(&b, "    %s\n", blocker)
			}
		case e.NotForTarget:
			fmt.Fprintf(&b, "  %s: %s\n", e.Name, i18n.T("UpdateAvailableButNotForTarget", i18n.MsgData{
				"Current": e.Name,
				"Latest":  e.Latest,
				"Target":  e.Target,
			}))
		case e.UpdateAvailable:
			b.WriteString(color.YellowString("  %s → %s", e.Name, e.Latest))
			b.WriteByte('\n')
		default:
			b.WriteString(color.GreenString("  %s ✓", e.Name))
			b.WriteByte('\n')
		}
	}
	fmt.Fprintf(&b, "\n  cjv %s\n", r.CjvVersion)
	if r.LatestCjvVersion != "" && r.LatestCjvVersion != r.CjvVersion {
		fmt.Fprintf(&b, "  cjv %s → %s\n", r.CjvVersion, r.LatestCjvVersion)
	}
	if r.SelfUpdateError != "" {
		fmt.Fprintf(&b, "  cjv: %s\n", r.SelfUpdateError)
	}
	if !r.HasUpdates && r.SelfUpdateError == "" && !slices.ContainsFunc(r.Toolchains, func(e checkEntry) bool { return e.Error != "" || e.NotForTarget || len(e.Blockers) > 0 }) {
		b.WriteString(color.GreenString(i18n.T("AllUpToDate", nil)))
		b.WriteByte('\n')
	}
	return b.String()
}

func (c *commands) runCheck(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()
	planned, planErr := lifecycle.PlanUpdates(ctx, nil, c.ctx.LifecycleOptions())
	if planErr != nil && len(planned.Toolchains) == 0 {
		return planErr
	}
	result := checkResult{CjvVersion: c.ctx.Version, NoneInstalled: planned.NoneInstalled}
	checked, checkErr := selfupdate.Check(ctx, c.ctx.UpdateURL, c.ctx.Version)
	if checkErr != nil {
		result.SelfUpdateError = checkErr.Error()
	} else if checked.Status == selfupdate.StatusAvailable || checked.Status == selfupdate.StatusUpToDate {
		result.LatestCjvVersion = checked.Version
		result.HasUpdates = checked.Status == selfupdate.StatusAvailable
	}
	for _, plan := range planned.Toolchains {
		if plan.Status == "skipped" {
			continue
		}
		result.Toolchains = append(result.Toolchains, checkEntry{
			Name: plan.Name, Current: plan.Current, Latest: plan.Latest, Selected: plan.Selected,
			Status: plan.Status, UpdateAvailable: plan.UpdateAvailable, Blockers: plan.Blockers, Error: plan.Error,
		})
		result.HasUpdates = result.HasUpdates || plan.UpdateAvailable
		for _, member := range plan.Members[min(1, len(plan.Members)):] {
			result.Toolchains = append(result.Toolchains, checkEntry{
				Name: member.Name, Current: member.Current, Latest: member.Selected, Selected: member.Selected,
				Status: plan.Status, UpdateAvailable: member.Removed || member.Selected != member.Current,
			})
		}
	}

	return c.ctx.Output.RenderTo(clictx.Out(cmd), result)
}

func (c *commands) checkCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "check",
		Short: i18n.T("CheckCmdShort", nil),
		RunE:  c.runCheck,
	}
}
