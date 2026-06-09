package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/output"
	componentlib "github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
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

	envCfg := env.LoadToolchainEnv(tcDir, componentlib.ApplyEnv)

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

	toolPath, found := resolveToolchainToolPath(tcDir, toolName)
	if !found {
		// Not a toolchain-local tool. Resolve it against the PATH we just built
		// (toolchain bin dirs prepended) instead of letting exec.Command
		// resolve a bare name against the parent process PATH — exec.LookPath
		// reads os.Getenv("PATH"), not c.Env, so the toolchain dirs would
		// otherwise be ignored.
		if resolved, ok := lookPathInEnv(toolName, procEnv); ok {
			toolPath = resolved
		}
	}

	c := exec.CommandContext(ctx, toolPath, toolArgs...)
	c.Env = procEnv
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	// Intercept termination signals before Start so a supervisor's SIGTERM in
	// the gap is buffered and forwarded to the child instead of killing only
	// the parent.
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
	if toolPath, err := proxy.ResolveInstalledToolBinary(tcDir, command); err == nil {
		return toolPath, true
	}
	binaryName := proxy.PlatformBinaryName(command)
	for _, subDir := range []string{"bin", filepath.Join("tools", "bin")} {
		candidate := filepath.Join(tcDir, subDir, binaryName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
	}
	return command, false
}

// lookPathInEnv resolves a bare command name against the PATH carried in
// environ, honoring PATHEXT on Windows. exec.Command resolves a bare name via
// exec.LookPath using the parent process PATH (os.Getenv), not c.Env, so this
// lets `cjv run` honor the toolchain bin dirs prepended to the child env.
// Returns the absolute path and true if found.
func lookPathInEnv(command string, environ []string) (string, bool) {
	if strings.ContainsRune(command, '/') || strings.ContainsRune(command, filepath.Separator) {
		return command, false // already a path; leave it for exec to handle
	}
	pathVal, _ := env.LookupValue(environ, "PATH")
	pathext, _ := env.LookupValue(environ, "PATHEXT")
	exts := executableExtensions(command, pathext)
	for _, dir := range filepath.SplitList(pathVal) {
		if dir == "" {
			continue
		}
		base := filepath.Join(dir, command)
		for _, ext := range exts {
			candidate := base + ext
			if isRegularExecutable(candidate) {
				return candidate, true
			}
		}
	}
	return command, false
}

// executableExtensions returns the suffixes to append to a bare command name
// when searching PATH. On non-Windows the only candidate is the name as given.
// On Windows, mirroring os/exec.LookPath: if the command already carries an
// extension it is used verbatim; otherwise each PATHEXT entry is tried and the
// bare extensionless name is NOT treated as executable — so a same-named data
// file cannot shadow the real .exe.
func executableExtensions(command, pathext string) []string {
	if runtime.GOOS != "windows" || filepath.Ext(command) != "" {
		return []string{""}
	}
	if pathext == "" {
		pathext = ".COM;.EXE;.BAT;.CMD"
	}
	var exts []string
	for p := range strings.SplitSeq(pathext, ";") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !strings.HasPrefix(p, ".") {
			p = "." + p
		}
		exts = append(exts, strings.ToLower(p))
	}
	return exts
}

// isRegularExecutable reports whether path is a regular file that is executable.
// On Windows executability is determined by extension (gated by the caller via
// executableExtensions), so any regular file matches here.
func isRegularExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode()&0o111 != 0
}

func init() {
	rootCmd.AddCommand(runCmd)
}
