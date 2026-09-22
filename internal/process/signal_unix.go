//go:build !windows

package process

import (
	"os"
	"syscall"
	"time"
)

// terminationGracePeriod bounds how long a child has to exit after being
// forwarded a SIGTERM before it is escalated to SIGKILL.
const terminationGracePeriod = 10 * time.Second

// SIGINT is intercepted so the parent does not exit before the child. The
// terminal already delivers it to the foreground process group. SIGTERM from
// a supervisor reaches cjv alone, so it must be forwarded to avoid orphans.
var terminationSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

func forwardSignal(proc *os.Process, sig os.Signal, done <-chan struct{}) {
	if proc == nil || sig != syscall.SIGTERM {
		return
	}
	_ = proc.Signal(syscall.SIGTERM) //nolint:errcheck // best-effort relay
	// Cancel escalation once the child has been reaped.
	go func() {
		select {
		case <-time.After(terminationGracePeriod):
			_ = proc.Kill() //nolint:errcheck // best-effort
		case <-done:
		}
	}()
}
