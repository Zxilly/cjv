package toolchaincmd

import (
	"errors"
	"strings"

	"charm.land/huh/v2"
	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// uninstall holds the flags of `cjv uninstall` and `cjv toolchain uninstall`.
type uninstall struct {
	ctx *clictx.Context
	yes bool
}

func newUninstall(ctx *clictx.Context) *uninstall {
	return &uninstall{ctx: ctx}
}

func (c *uninstall) run(cmd *cobra.Command, args []string) error {
	var errs []error
	var removed []uninstallResult
	for _, name := range args {
		result, err := c.runOne(name)
		if err != nil {
			errs = append(errs, err)
		} else if result != nil {
			removed = append(removed, *result)
		}
	}
	if len(args) == 1 && len(removed) == 1 {
		return c.ctx.Output.RenderOutcome(clictx.Out(cmd), removed[0], errors.Join(errs...))
	}
	return c.ctx.Output.RenderOutcome(clictx.Out(cmd), uninstallBatchResult{Removed: removed}, errors.Join(errs...))
}

func (c *uninstall) runOne(name string) (*uninstallResult, error) {
	if err := lifecycle.PrepareToolchainRemoval(name); err != nil {
		return nil, err
	}

	// Confirm before destroying the toolchain and its components. Skipped with
	// --yes, in JSON mode, or when stdin is not an interactive terminal (so
	// scripts/CI are not blocked waiting on a prompt).
	if !c.yes && !c.ctx.Output.IsJSON() && clictx.StdinIsTerminal() {
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

func (c *uninstall) command(use, short string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   use,
		Short: short,
		Args:  cobra.MinimumNArgs(1),
		RunE:  c.run,
	}
	cmd.Flags().BoolVarP(&c.yes, "yes", "y", false, i18n.T("FlagSkipConfirm", nil))
	return cmd
}
