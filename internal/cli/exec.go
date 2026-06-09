package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/output"
	componentlib "github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/env"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:                "exec [+toolchain] <command> [args...]",
	Short:              i18n.T("ExecCmdShort", nil),
	Long:               i18n.T("ExecCmdLong", nil),
	Args:               cobra.ArbitraryArgs,
	RunE:               execRun,
	DisableFlagParsing: true,
}

// extractPlusToolchainFromArgs extracts an optional +toolchain prefix from args
// (used by envsetup, which lets cobra parse flags). A bare "+" is ignored.
func extractPlusToolchainFromArgs(args []string) (string, []string) {
	if tc, rest, present := toolchain.SplitPlusSelector(args); present && tc != "" {
		return tc, rest
	}
	return "", args
}

func execRun(cmd *cobra.Command, args []string) error {
	tcOverride, remaining, err := stripJSONModeFlagPrefix(args, true)
	if err != nil {
		return err
	}
	if output.IsJSON() {
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
		return fmt.Errorf("requires at least 1 argument: <command>")
	}

	command := remaining[0]
	commandArgs := remaining[1:]

	runtimeEnv, err := env.ResolveRuntimeEnv(ctx, tcOverride, componentlib.ApplyEnv)
	if err != nil {
		return err
	}

	// Resolve a bare command against the runtime env's PATH (toolchain bin dirs
	// prepended) rather than letting exec.Command resolve it against the parent
	// process PATH — exec.LookPath reads os.Getenv("PATH"), not c.Env, so a tool
	// present only in the toolchain would otherwise not be found (mirrors run.go).
	if resolved, ok := lookPathInEnv(command, runtimeEnv); ok {
		command = resolved
	}

	c := exec.CommandContext(ctx, command, commandArgs...)
	c.Env = runtimeEnv
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Start(); err != nil {
		return err
	}
	stopForward := proxy.ForwardTerminationSignals(c.Process)
	defer stopForward()

	if err := c.Wait(); err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			return &cjverr.ExitCodeError{Code: exitErr.ExitCode()}
		}
		return err
	}
	return nil
}

func init() {
	rootCmd.AddCommand(execCmd)
}
