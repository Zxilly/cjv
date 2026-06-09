package proxy

import (
	"os"
	"os/signal"
)

// ForwardTerminationSignals keeps the parent process alive long enough to
// collect the child's exit status and relays termination signals to the child
// so a supervisor's SIGTERM reaches the spawned tool instead of orphaning it.
//
// Ctrl+C / SIGINT is intercepted only to stop the parent from dying first; the
// terminal already delivers it to the child via the shared foreground process
// group, so it is not forwarded again (that would deliver it twice). The exact
// signal set and per-signal forwarding policy are platform specific — see
// signal_unix.go / signal_windows.go.
//
// The returned function stops the handler and must be called once the child
// has been waited on.
func ForwardTerminationSignals(proc *os.Process) func() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, terminationSignals...)

	done := make(chan struct{})
	go func() {
		for {
			select {
			case sig := <-sigCh:
				forwardSignal(proc, sig, done)
			case <-done:
				return
			}
		}
	}()

	return func() {
		close(done)
		signal.Stop(sigCh)
	}
}
