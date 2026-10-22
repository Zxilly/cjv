package toolchaincmd

import (
	"fmt"
	"strings"

	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/spf13/cobra"
)

type historyListResult struct {
	SchemaVersion int                      `json:"schema_version"`
	Snapshots     []lifecycle.HistoryEntry `json:"snapshots"`
}

func (r historyListResult) Text() string {
	if len(r.Snapshots) == 0 {
		return "No retained toolchain history."
	}
	var lines []string
	for _, snapshot := range r.Snapshots {
		lines = append(lines, fmt.Sprintf("%s  %s  %s  (%d members)", snapshot.Toolchain, snapshot.ID, snapshot.Members[0].Release, len(snapshot.Members)))
	}
	return strings.Join(lines, "\n")
}

type historyPruneResult struct {
	SchemaVersion int `json:"schema_version"`
	Removed       int `json:"removed"`
	Keep          int `json:"keep"`
}

func (r historyPruneResult) Text() string {
	return fmt.Sprintf("Removed %d history snapshots; retained at most %d per toolchain.", r.Removed, r.Keep)
}

type historyRollbackResult struct{ lifecycle.RollbackResult }

func (r historyRollbackResult) Text() string {
	return fmt.Sprintf("Restored %s from %s (%d members).", r.Toolchain, r.Snapshot, len(r.Members))
}

func historyCommand(ctx *clictx.Context) *cobra.Command {
	runList := func(cmd *cobra.Command, args []string) error {
		name := ""
		if len(args) > 0 {
			name = args[0]
		}
		entries, err := lifecycle.ListHistory(cmd.Context(), name)
		if err != nil {
			return err
		}
		return ctx.Output.RenderTo(clictx.Out(cmd), historyListResult{SchemaVersion: 1, Snapshots: entries})
	}
	cmd := &cobra.Command{Use: "history [toolchain]", Short: "List retained tracking toolchain snapshots", Args: cobra.MaximumNArgs(1), RunE: runList}
	cmd.AddCommand(&cobra.Command{Use: "list [toolchain]", Short: "List retained tracking toolchain snapshots", Args: cobra.MaximumNArgs(1), RunE: runList})
	keep := lifecycle.DefaultHistoryRetention
	prune := &cobra.Command{Use: "prune [toolchain]", Short: "Remove older retained toolchain snapshots", Args: cobra.MaximumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
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
	prune.Flags().IntVar(&keep, "keep", lifecycle.DefaultHistoryRetention, "Number of recent snapshots to retain per tracking toolchain")
	cmd.AddCommand(prune)
	return cmd
}

func rollbackCommand(ctx *clictx.Context) *cobra.Command {
	return &cobra.Command{Use: "rollback <toolchain> [snapshot]", Short: "Restore a retained tracking toolchain group", Args: cobra.RangeArgs(1, 2), RunE: func(cmd *cobra.Command, args []string) error {
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
