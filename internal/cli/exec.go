package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/output"
	componentlib "github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/env"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:                "exec [+toolchain] <command> [args...]",
	Short:              "Run a command with Cangjie runtime environment",
	Long:               "Execute any command with the correct Cangjie runtime library paths injected.",
	Args:               cobra.ArbitraryArgs,
	RunE:               execRun,
	DisableFlagParsing: true,
}

// extractPlusToolchainFromArgs extracts an optional +toolchain prefix from args.
func extractPlusToolchainFromArgs(args []string) (string, []string) {
	if len(args) > 0 && strings.HasPrefix(args[0], "+") {
		tc := args[0][1:]
		if tc != "" {
			return tc, args[1:]
		}
	}
	return "", args
}

func execRun(cmd *cobra.Command, args []string) error {
	var err error
	args, err = stripJSONModeFlagPrefix(args, true)
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

	if len(args) > 0 && (args[0] == "--help" || args[0] == "-h") {
		return cmd.Help()
	}

	tcOverride, remaining := extractPlusToolchainFromArgs(args)

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

	stopIntercept := proxy.InterceptInterrupts()
	defer stopIntercept()

	c := exec.CommandContext(ctx, command, commandArgs...)
	c.Env = runtimeEnv
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
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
