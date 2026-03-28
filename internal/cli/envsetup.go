package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Zxilly/cjv/internal/env"
	"github.com/spf13/cobra"
)

var envsetupCmd = &cobra.Command{
	Use:   "envsetup [+toolchain] [--shell=TYPE]",
	Short: "Print shell commands to configure Cangjie runtime environment",
	Long: `Output shell commands that set environment variables for the active Cangjie toolchain.

Usage:
  eval "$(cjv envsetup)"          # bash/zsh
  cjv envsetup | source           # fish
  cjv envsetup | Invoke-Expression  # powershell`,
	Args:               cobra.ArbitraryArgs,
	RunE:               envsetupRun,
	DisableFlagParsing: true,
}

func envsetupRun(cmd *cobra.Command, args []string) error {
	// Manual flag parsing since DisableFlagParsing is true.
	shellFlag := ""
	var remaining []string
	for _, a := range args {
		switch {
		case a == "--help" || a == "-h":
			return cmd.Help()
		case strings.HasPrefix(a, "--shell="):
			shellFlag = strings.TrimPrefix(a, "--shell=")
		default:
			remaining = append(remaining, a)
		}
	}
	return envsetupRunWithShell(cmd, remaining, shellFlag)
}

func envsetupRunWithShell(cmd *cobra.Command, args []string, shellFlag string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	tcOverride, _ := extractPlusToolchainFromArgs(args)

	// Determine shell type
	var shellType env.ShellType
	if shellFlag != "" {
		st, err := env.ParseShellFlag(shellFlag)
		if err != nil {
			return err
		}
		shellType = st
	} else {
		st, detected := env.DetectShell()
		if !detected {
			fmt.Fprintln(os.Stderr, "cjv: could not detect shell type, defaulting to posix. Use --shell=TYPE to override (bash, fish, powershell, cmd)")
		}
		shellType = st
	}

	baseEnv := os.Environ()

	runtimeEnv, err := env.ResolveRuntimeEnv(ctx, tcOverride)
	if err != nil {
		return err
	}

	diff := env.ComputeEnvDiff(baseEnv, runtimeEnv)
	if len(diff) == 0 {
		return nil
	}

	output := env.FormatEnvDiff(diff, shellType)
	fmt.Fprint(cmd.OutOrStdout(), output)
	return nil
}

func init() {
	rootCmd.AddCommand(envsetupCmd)
}
