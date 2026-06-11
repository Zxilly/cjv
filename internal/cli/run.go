package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/output"
	componentlib "github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/env"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run [--install] <toolchain> <command> [args...]",
	Short: i18n.T("RunCmdShort", nil),
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
		matched, err := applyJSONModeFlag(args[0])
		if err != nil {
			return err
		}
		if matched {
			args = args[1:]
			continue
		}
		switch args[0] {
		case "--install":
			install = true
			args = args[1:]
		case "--help", "-h":
			return cmd.Help()
		case "--":
			args = args[1:]
			break flagLoop
		default:
			break flagLoop
		}
	}
	if output.IsJSON() {
		return &cjverr.UnsupportedForJSONError{Command: "run"}
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

	count := proxy.GetRecursionCount()

	rt, err := env.RuntimeForToolchain(tcDir, filepath.Base(tcDir), componentlib.ApplyEnv)
	if err != nil {
		return err
	}

	procEnv := rt.ProxyEnv(os.Environ(), count)

	toolPath, found := resolveToolchainToolPath(tcDir, toolName)
	if !found {
		// Not a toolchain-local tool. Resolve it against the PATH we just built
		// (toolchain bin dirs prepended) instead of letting exec.Command
		// resolve a bare name against the parent process PATH — exec.LookPath
		// reads os.Getenv("PATH"), not c.Env, so the toolchain dirs would
		// otherwise be ignored.
		if resolved, ok := env.LookPathInEnv(toolName, procEnv); ok {
			toolPath = resolved
		}
	}

	c := exec.CommandContext(ctx, toolPath, toolArgs...)
	c.Env = procEnv
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	forwarder := proxy.NewTerminationForwarder()
	defer forwarder.Stop()
	if err := c.Start(); err != nil {
		return err
	}
	forwarder.Attach(c.Process)

	if err := c.Wait(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return &cjverr.ExitCodeError{Code: exitErr.ExitCode()}
		}
		return err
	}
	return nil
}

// resolveToolchainToolPath resolves command to an absolute path within tcDir,
// first consulting the known proxy-tool table, then scanning bin/ and
// tools/bin/. The bool reports whether the tool was found inside the toolchain;
// when false the returned path is the bare command name.
func resolveToolchainToolPath(tcDir, command string) (string, bool) {
	return env.ResolveToolPath(tcDir, command, proxy.ResolveInstalledToolBinary, proxy.PlatformBinaryName)
}

// lookPathInEnv resolves a bare command name against the PATH carried in
// environ, honoring PATHEXT on Windows. exec.Command resolves a bare name via
// exec.LookPath using the parent process PATH (os.Getenv), not c.Env, so this
// lets `cjv run` honor the toolchain bin dirs prepended to the child env.
// Returns the absolute path and true if found.
func lookPathInEnv(command string, environ []string) (string, bool) {
	return env.LookPathInEnv(command, environ)
}

func init() {
	rootCmd.AddCommand(runCmd)
}
