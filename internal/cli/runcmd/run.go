package runcmd

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/env"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/process"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
)

func (c *commands) runRun(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Manual flag parsing since DisableFlagParsing is true.
	// Consume all flags before the positional <toolchain> argument.
	install := false
flagLoop:
	for len(args) > 0 {
		matched, err := c.ctx.ApplyJSONModeFlag(args[0])
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
	if c.ctx.Output.IsJSON() {
		return &cjverr.UnsupportedForJSONError{Command: "run"}
	}

	if len(args) < 2 {
		_ = cmd.Help()
		return fmt.Errorf("requires at least 2 arguments: <toolchain> <command>")
	}

	tcInput := args[0]
	toolName := args[1]
	toolArgs := args[2:]

	var installMissing func(context.Context) error
	if install {
		installMissing = func(ctx context.Context) error {
			return lifecycle.Install(ctx, lifecycle.InstallRequest{Toolchain: tcInput}, c.ctx.LifecycleOptions())
		}
	}
	tcDir, _, err := toolchain.PrepareActive(ctx, tcInput, installMissing)
	if err != nil {
		return err
	}

	count := proxy.GetRecursionCount()

	rt, err := env.RuntimeForToolchain(tcDir, filepath.Base(tcDir))
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

	child := exec.CommandContext(ctx, toolPath, toolArgs...)
	child.Env = procEnv
	child.Stdin = os.Stdin
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr

	return process.Run(child)
}

// resolveToolchainToolPath resolves command to an absolute path within tcDir,
// first consulting the known proxy-tool table, then scanning bin/ and
// tools/bin/. The bool reports whether the tool was found inside the toolchain;
// when false the returned path is the bare command name.
func resolveToolchainToolPath(tcDir, command string) (string, bool) {
	return env.ResolveToolPath(tcDir, command)
}
