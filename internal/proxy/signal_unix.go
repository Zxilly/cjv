//go:build !windows

package proxy

import (
	"os"
	"syscall"
	"time"
)

// terminationGracePeriod bounds how long a child has to exit after being
// forwarded a SIGTERM before it is escalated to SIGKILL.
const terminationGracePeriod = 10 * time.Second

// terminationSignals are the signals the parent intercepts while a child runs.
// SIGINT (Ctrl+C) is intercepted so the parent does not exit before the child;
// the terminal already delivers it to the child via the foreground process
// group, so it is not forwarded again. SIGTERM is forwarded because a
// supervisor (systemd, `kill <pid>`) sends it to the cjv process alone, not the
// whole group, so without forwarding the child would be orphaned.
var terminationSignals = []os.Signal{os.Interrupt, syscall.SIGTERM}

func forwardSignal(proc *os.Process, sig os.Signal, done <-chan struct{}) {
	if proc == nil || sig != syscall.SIGTERM {
		return
	}
	_ = proc.Signal(syscall.SIGTERM) //nolint:errcheck // best-effort relay
	// Escalate to SIGKILL if the child neither handles SIGTERM nor exits within
	// the grace period, so an unresponsive child cannot hang cjv (and the
	// supervisor) indefinitely. done is closed once the child has been reaped,
	// which cancels the escalation before it fires.
	go func() {
		select {
		case <-time.After(terminationGracePeriod):
			_ = proc.Kill() //nolint:errcheck // best-effort
		case <-done:
		}
	}()
}
