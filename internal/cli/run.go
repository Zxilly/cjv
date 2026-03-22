package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/env"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [--install] <toolchain> <command> [args...]",
	Short: "Run a command with a specific toolchain",
	// Use ArbitraryArgs because DisableFlagParsing treats all tokens
	// (including --help, --install) as positional args, making MinimumNArgs
	// reject valid inputs like "cjv run --help". Validation is done in runRun.
	Args:               cobra.ArbitraryArgs,
	RunE:               runRun,
	DisableFlagParsing: true,
}

func runRun(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Manual flag parsing since DisableFlagParsing is true.
	// Consume all flags before the positional <toolchain> argument.
	install := false
flagLoop:
	for len(args) > 0 {
		switch args[0] {
		case "--install":
			install = true
			args = args[1:]
		case "--help", "-h":
			return cmd.Help()
		default:
			break flagLoop
		}
	}

	if len(args) < 2 {
		_ = cmd.Help()
		return fmt.Errorf("requires at least 2 arguments: <toolchain> <command>")
	}

	tcInput := args[0]
	toolName := args[1]
	toolArgs := args[2:]

	parsed, err := toolchain.ParseToolchainName(tcInput)
	if err != nil {
		return err
	}

	tcDir, findErr := toolchain.FindInstalled(parsed)
	if findErr != nil {
		if install {
			if installErr := InstallToolchainWithOptions(ctx, tcInput, false); installErr != nil {
				return installErr
			}
			tcDir, findErr = toolchain.FindInstalled(parsed)
			if findErr != nil {
				return &cjverr.ToolchainNotInstalledError{Name: tcInput}
			}
		} else {
			return &cjverr.ToolchainNotInstalledError{Name: tcInput}
		}
	}

	toolPath := resolveRunCommand(tcDir, toolName)

	envCfg := env.LoadToolchainEnv(ctx, tcDir)

	count := proxy.GetRecursionCount()

	// Get cjv bin dir to remove from PATH (prevent proxy recursion)
	binDir, err := config.BinDir()
	if err != nil {
		return fmt.Errorf("failed to determine bin directory: %w", err)
	}

	procEnv := env.BuildProxyEnv(os.Environ(), env.ProxyEnvContext{
		Cfg:             envCfg,
		CjvBinDir:       binDir,
		ToolchainBinDir: filepath.Join(tcDir, "bin"),
		Recursion:       count,
		ToolchainName:   filepath.Base(tcDir),
	})

	stopIntercept := proxy.InterceptInterrupts()
	defer stopIntercept()

	c := exec.CommandContext(ctx, toolPath, toolArgs...)
	c.Env = procEnv
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	if err := c.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return &cjverr.ExitCodeError{Code: exitErr.ExitCode()}
		}
		return err
	}
	return nil
}

// resolveRunCommand prefers a toolchain binary when it exists, otherwise falls
// back to the original command name and lets PATH resolve it.
func resolveRunCommand(tcDir, command string) string {
	// First try known proxy tools (bin/ and tools/bin/)
	if toolPath, err := proxy.ResolveInstalledToolBinary(tcDir, command); err == nil {
		return toolPath
	}
	binaryName := proxy.PlatformBinaryName(command)
	for _, subDir := range []string{"bin", filepath.Join("tools", "bin")} {
		candidate := filepath.Join(tcDir, subDir, binaryName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
	}
	return command
}

func init() {
	rootCmd.AddCommand(runCmd)
}
