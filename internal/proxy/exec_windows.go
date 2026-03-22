//go:build windows

package proxy

import (
	"context"
	"errors"
	"os"
	"os/exec"

	"github.com/Zxilly/cjv/internal/cjverr"
)

// execTool runs the tool as a child process and propagates the exit code
// via ExitCodeError so that deferred cleanup functions can still run.
func execTool(ctx context.Context, binary string, args []string, env []string) error {
	stop := InterceptInterrupts()
	defer stop()

	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			return &cjverr.ExitCodeError{Code: exitErr.ExitCode()}
		}
		return err
	}
	return nil
}
