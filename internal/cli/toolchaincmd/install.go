// Package toolchaincmd implements the commands that change installed
// toolchains: install, uninstall, update, toolchain, and target.
package toolchaincmd

import (
	"errors"

	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/cli/output"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/spf13/cobra"
)

// Register adds install, uninstall, update, toolchain, and target to root.
func Register(root *cobra.Command, ctx *clictx.Context) {
	root.AddCommand(
		newInstall(ctx).command(),
		newUninstall(ctx).command("uninstall <toolchain>...", i18n.T("UninstallCmdShort", nil)),
		newUpdate(ctx).command(),
		toolchainCommand(ctx),
		targetCommand(ctx),
	)
}

type installResult struct {
	output.ProgressDriven
	Toolchain  string   `json:"toolchain"`
	Targets    []string `json:"targets"`
	Components []string `json:"components"`
	Forced     bool     `json:"forced"`
}

type installBatchResult struct {
	output.ProgressDriven
	Installations []installResult `json:"installations"`
}

// install holds the flags of `cjv install` for one invocation.
type install struct {
	ctx            *clictx.Context
	force          bool
	targets        []string
	components     []string
	noUpdate       bool
	allowDowngrade bool
}

func newInstall(ctx *clictx.Context) *install {
	return &install{ctx: ctx}
}

func (c *install) lifecycleOptions() lifecycle.Options {
	opts := c.ctx.LifecycleOptions()
	opts.AllowMissing = c.force
	opts.AllowDowngrade = c.allowDowngrade
	return opts
}

func (c *install) run(cmd *cobra.Command, args []string) error {
	clictx.CheckSudoSafety()
	if len(args) == 0 {
		name, err := c.ctx.SelectedToolchain("")
		if err != nil {
			return err
		}
		args = []string{name}
	}
	var errs []error
	var results []installResult
	for _, input := range args {
		err := lifecycle.Install(cmd.Context(), lifecycle.InstallRequest{
			Toolchain:  input,
			Targets:    c.targets,
			Components: c.components,
			Force:      c.force,
			NoUpdate:   c.noUpdate,
		}, c.lifecycleOptions())
		if err != nil {
			errs = append(errs, err)
		} else {
			results = append(results, installResult{Toolchain: input, Targets: c.targets, Components: c.components, Forced: c.force})
		}
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}
	if len(args) > 1 {
		return c.ctx.Output.RenderTo(clictx.Out(cmd), installBatchResult{Installations: results})
	}
	return c.ctx.Output.RenderTo(clictx.Out(cmd), installResult{
		Toolchain:  args[0],
		Targets:    c.targets,
		Components: c.components,
		Forced:     c.force,
	})
}

func (c *install) command() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "install [toolchain]...",
		Short: i18n.T("InstallCmdShort", nil),
		Args:  cobra.ArbitraryArgs,
		RunE:  c.run,
	}
	cmd.Flags().BoolVar(&c.force, "force", false, i18n.T("InstallFlagForce", nil))
	cmd.Flags().StringSliceVarP(&c.targets, "target", "t", nil, i18n.T("InstallFlagTarget", nil))
	cmd.Flags().StringSliceVarP(&c.components, "component", "c", nil, i18n.T("InstallFlagComponent", nil))
	cmd.Flags().BoolVar(&c.noUpdate, "no-update", false, "Keep an existing SDK release while adding targets or components")
	cmd.Flags().BoolVar(&c.allowDowngrade, "allow-downgrade", false, "Allow an older nightly that supplies the requested components")
	cmd.MarkFlagsMutuallyExclusive("force", "no-update")
	return cmd
}
