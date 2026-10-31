package runcmd

import (
	"context"
	"errors"
	"os"
	"os/exec"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/env"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/process"
	"github.com/Zxilly/cjv/internal/resolve"
	"github.com/spf13/cobra"
)

func (c *commands) execRun(cmd *cobra.Command, args []string) error {
	defer c.ctx.RestoreEnvironment()
	tcOverride, remaining, err := c.ctx.StripGlobalFlagPrefix(args, true)
	if err != nil {
		return err
	}
	if tcOverride == "" {
		tcOverride = c.ctx.Selector
	}
	if err := c.ctx.ApplyInvocation(cmd); err != nil {
		return err
	}
	if c.ctx.Output.IsJSON() {
		return &cjverr.UnsupportedForJSONError{Command: "exec"}
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if len(remaining) > 0 && (remaining[0] == "--help" || remaining[0] == "-h") {
		return cmd.Help()
	}

	if len(remaining) == 0 {
		_ = cmd.Help()
		return errors.New(i18n.T("ExecMissingCommand", nil))
	}

	command := remaining[0]
	commandArgs := remaining[1:]

	runtimeEnv, err := resolve.ActiveRuntimeEnv(ctx, tcOverride)
	if err != nil {
		return err
	}

	// Resolve a bare command against the runtime env's PATH (toolchain bin dirs
	// prepended) rather than letting exec.Command resolve it against the parent
	// process PATH — exec.LookPath reads os.Getenv("PATH"), not c.Env, so a tool
	// present only in the toolchain would otherwise not be found (mirrors run.go).
	if resolved, ok := env.LookPathInEnv(command, runtimeEnv); ok {
		command = resolved
	}

	child := exec.CommandContext(ctx, command, commandArgs...)
	child.Env = runtimeEnv
	child.Stdin = os.Stdin
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr

	return process.Run(child)
}
