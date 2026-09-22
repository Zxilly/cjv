//go:build windows

package proxy

import (
	"context"
	"os"
	"os/exec"

	"github.com/Zxilly/cjv/internal/process"
)

// execTool runs the tool as a child process and propagates the exit code
// via ExitCodeError so that deferred cleanup functions can still run.
func execTool(ctx context.Context, binary string, args []string, env []string) error {
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return process.Run(cmd)
}
