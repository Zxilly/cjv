package cli

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/output"
	componentlib "github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/env"
	"github.com/spf13/cobra"
)

const jsonFlagName = "json"

func parseJSONModeFlag(arg string) (bool, bool, error) {
	if arg == "--"+jsonFlagName {
		return true, true, nil
	}
	prefix := "--" + jsonFlagName + "="
	if !strings.HasPrefix(arg, prefix) {
		return false, false, nil
	}
	value, err := strconv.ParseBool(strings.TrimPrefix(arg, prefix))
	if err != nil {
		return true, false, fmt.Errorf("invalid --%s value %q", jsonFlagName, strings.TrimPrefix(arg, prefix))
	}
	return true, value, nil
}

func applyJSONModeFlag(arg string) (bool, error) {
	matched, value, err := parseJSONModeFlag(arg)
	if err != nil || !matched {
		return matched, err
	}
	output.SetJSONMode(value)
	return true, nil
}

func stripJSONModeFlags(args []string) ([]string, error) {
	out := make([]string, 0, len(args))
	for _, arg := range args {
		matched, err := applyJSONModeFlag(arg)
		if err != nil {
			return nil, err
		}
		if matched {
			continue
		}
		out = append(out, arg)
	}
	return out, nil
}

func stripJSONModeFlagPrefix(args []string, allowAfterPlusToolchain bool) ([]string, error) {
	out := make([]string, 0, len(args))
	scanning := true
	sawPlusToolchain := false
	for i, arg := range args {
		if scanning {
			if arg == "--" {
				out = append(out, args[i+1:]...)
				return out, nil
			}
			matched, err := applyJSONModeFlag(arg)
			if err != nil {
				return nil, err
			}
			if matched {
				continue
			}
			if allowAfterPlusToolchain && !sawPlusToolchain && strings.HasPrefix(arg, "+") && len(arg) > 1 {
				sawPlusToolchain = true
				out = append(out, arg)
				continue
			}
			scanning = false
		}
		out = append(out, arg)
	}
	return out, nil
}

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
	var err error
	args, err = stripJSONModeFlags(args)
	if err != nil {
		return err
	}
	if output.IsJSON() {
		return &cjverr.UnsupportedForJSONError{Command: "envsetup"}
	}
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

	runtimeEnv, err := env.ResolveRuntimeEnv(ctx, tcOverride, componentlib.ApplyEnv)
	if err != nil {
		return err
	}

	diff := env.ComputeEnvDiff(baseEnv, runtimeEnv)
	if len(diff) == 0 {
		return nil
	}

	output := env.FormatEnvDiff(diff, shellType)
	_, _ = fmt.Fprint(cmdOutput(cmd), output)
	return nil
}

func init() {
	rootCmd.AddCommand(envsetupCmd)
}
