// Package runcmd implements the commands that run tools or emit the SDK
// environment: run, exec, which, and envsetup.
package runcmd

import (
	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/spf13/cobra"
)

// commands holds the state of the run commands for one invocation. run and
// exec parse their own arguments, so only which has a cobra flag.
type commands struct {
	ctx            *clictx.Context
	whichToolchain string
}

// Register adds run, exec, which, and envsetup to root.
func Register(root *cobra.Command, ctx *clictx.Context) {
	c := &commands{ctx: ctx}
	root.AddCommand(c.runCommand(), c.execCommand(), c.whichCommand(), c.newEnvsetupCmd())
}

func (c *commands) runCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "run [--install] <toolchain> <command> [args...]",
		Short: i18n.T("RunCmdShort", nil),
		// Use ArbitraryArgs because DisableFlagParsing treats all tokens
		// (including --help, --install) as positional args, making MinimumNArgs
		// reject valid inputs like "cjv run --help". Validation is done in runRun.
		Args:               cobra.ArbitraryArgs,
		RunE:               c.runRun,
		DisableFlagParsing: true,
	}
}

func (c *commands) execCommand() *cobra.Command {
	return &cobra.Command{
		Use:                "exec [+toolchain] <command> [args...]",
		Short:              i18n.T("ExecCmdShort", nil),
		Long:               i18n.T("ExecCmdLong", nil),
		Args:               cobra.ArbitraryArgs,
		RunE:               c.execRun,
		DisableFlagParsing: true,
	}
}

func (c *commands) whichCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "which [command]",
		Short: i18n.T("WhichCmdShort", nil),
		Long:  i18n.T("WhichCmdLong", nil),
		Args:  cobra.MaximumNArgs(1),
		RunE:  c.runWhich,
	}
	cmd.Flags().StringVar(&c.whichToolchain, "toolchain", "", "Select the toolchain")
	return cmd
}
