// Package process runs child processes while preserving their exit status and
// relaying the termination signals received by cjv.
package process

import (
	"errors"
	"os/exec"

	"github.com/Zxilly/cjv/internal/cjverr"
)

// Run starts an unstarted command and waits for it to finish. The caller
// configures its context, environment and standard streams as with exec.Cmd.
// Nonzero child exits are returned as ExitCodeError; start and other wait
// errors are preserved.
func Run(cmd *exec.Cmd) error {
	forwarder := newTerminationForwarder()
	defer forwarder.stop()
	if err := cmd.Start(); err != nil {
		return err
	}
	forwarder.attach(cmd.Process)

	if err := cmd.Wait(); err != nil {
		if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
			return &cjverr.ExitCodeError{Code: exitErr.ExitCode()}
		}
		return err
	}
	return nil
}
