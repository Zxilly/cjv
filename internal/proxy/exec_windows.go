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
	cmd := exec.CommandContext(ctx, binary, args...)
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Intercept termination signals before Start so a signal in the gap is
	// buffered and forwarded to the child instead of killing only the parent.
	forwarder := NewTerminationForwarder()
	defer forwarder.Stop()
	if err := cmd.Start(); err != nil {
		return err
	}
	forwarder.Attach(cmd.Process)

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			return &cjverr.ExitCodeError{Code: exitErr.ExitCode()}
		}
		return err
	}
	return nil
}
