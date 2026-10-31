package toolchaincmd

import (
	"fmt"
	"strings"

	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/spf13/cobra"
)

type historyListResult struct {
	SchemaVersion int `json:"schema_version"`
	lifecycle.HistoryReport
}

func (r historyListResult) Text() string {
	if len(r.Snapshots) == 0 && len(r.Invalid) == 0 {
		return i18n.T("HistoryEmpty", nil)
	}
	var lines []string
	for _, snapshot := range r.Snapshots {
		lines = append(lines, fmt.Sprintf(i18n.T("HistorySnapshot", nil), snapshot.Toolchain, snapshot.ID, snapshot.Members[0].Release, len(snapshot.Members)))
	}
	for _, invalid := range r.Invalid {
		lines = append(lines, fmt.Sprintf(i18n.T("HistoryInvalid", nil), invalid.Toolchain, invalid.ID, invalid.Error))
	}
	if len(r.Invalid) > 0 {
		lines = append(lines, i18n.T("HistoryPruneHint", nil))
	}
	return strings.Join(lines, "\n")
}

type historyPruneResult struct {
	SchemaVersion int `json:"schema_version"`
	Removed       int `json:"removed"`
	Keep          int `json:"keep"`
}

func (r historyPruneResult) Text() string {
	return fmt.Sprintf(i18n.T("HistoryPruned", nil), r.Removed, r.Keep)
}

type historyRollbackResult struct{ lifecycle.RollbackResult }

func (r historyRollbackResult) Text() string {
	return fmt.Sprintf(i18n.T("HistoryRestored", nil), r.Toolchain, r.Snapshot, len(r.Members))
}

func historyCommand(ctx *clictx.Context) *cobra.Command {
	runList := func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) > 0 {
			name = args[0]
		}
		report, err := lifecycle.InspectHistory(cmd.Context(), name)
		if err != nil {
			return err
		}
		return ctx.Output.RenderTo(clictx.Out(cmd), historyListResult{SchemaVersion: 1, HistoryReport: report})
	}
	cmd := &cobra.Command{Use: "history [toolchain]", Short: i18n.T("HistoryListShort", nil), Args: cobra.MaximumNArgs(1), RunE: runList}
	cmd.AddCommand(&cobra.Command{Use: "list [toolchain]", Short: i18n.T("HistoryListShort", nil), Args: cobra.MaximumNArgs(1), RunE: runList})
	keep := lifecycle.DefaultHistoryRetention
	prune := &cobra.Command{Use: "prune [toolchain]", Short: i18n.T("HistoryPruneShort", nil), Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) > 0 {
			name = args[0]
		}
		removed, err := lifecycle.PruneHistory(cmd.Context(), name, keep)
		if err != nil {
			return err
		}
		return ctx.Output.RenderTo(clictx.Out(cmd), historyPruneResult{SchemaVersion: 1, Removed: removed, Keep: keep})
	}}
	prune.Flags().IntVar(&keep, "keep", lifecycle.DefaultHistoryRetention, i18n.T("HistoryFlagKeep", nil))
	cmd.AddCommand(prune)
	return cmd
}

func rollbackCommand(ctx *clictx.Context) *cobra.Command {
	return &cobra.Command{Use: "rollback <toolchain> [snapshot]", Short: i18n.T("RollbackCmdShort", nil), Args: cobra.RangeArgs(1, 2), RunE: func(cmd *cobra.Command, args []string) error {
		id := ""
		if len(args) > 1 {
			id = args[1]
		}
		result, err := lifecycle.RollbackHistory(cmd.Context(), args[0], id, ctx.LifecycleOptions())
		if err != nil {
			return err
		}
		return ctx.Output.RenderTo(clictx.Out(cmd), historyRollbackResult{result})
	}}
}
