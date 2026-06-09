package proxy

import (
	"os"
	"os/signal"
)

// TerminationForwarder keeps the parent process alive long enough to collect
// the child's exit status and relays termination signals to the child so a
// supervisor's SIGTERM reaches the spawned tool instead of orphaning it.
//
// Interception starts at construction, before the child is spawned: a signal
// landing between Start and Attach would otherwise hit the parent's default
// disposition, killing it without ever reaching the child. Such signals are
// buffered and replayed once Attach supplies the process handle.
//
// Ctrl+C / SIGINT is intercepted only to stop the parent from dying first; the
// terminal already delivers it to the child via the shared foreground process
// group, so it is not forwarded again (that would deliver it twice). The exact
// signal set and per-signal forwarding policy are platform specific — see
// signal_unix.go / signal_windows.go.
type TerminationForwarder struct {
	sigCh  chan os.Signal
	procCh chan *os.Process
	done   chan struct{}
}

// NewTerminationForwarder installs the signal handler and starts buffering.
// Call it before starting the child, Attach once the child is running, and
// Stop exactly once after the child has been waited on (or the start failed).
func NewTerminationForwarder() *TerminationForwarder {
	f := &TerminationForwarder{
		sigCh:  make(chan os.Signal, 1),
		procCh: make(chan *os.Process, 1),
		done:   make(chan struct{}),
	}
	signal.Notify(f.sigCh, terminationSignals...)
	go f.run()
	return f
}

func (f *TerminationForwarder) run() {
	var proc *os.Process
	var pending []os.Signal
	for {
		select {
		case proc = <-f.procCh:
			for _, sig := range pending {
				forwardSignal(proc, sig, f.done)
			}
			pending = nil
		case sig := <-f.sigCh:
			if proc == nil {
				pending = append(pending, sig)
				continue
			}
			forwardSignal(proc, sig, f.done)
		case <-f.done:
			return
		}
	}
}

// Attach hands the started child to the forwarder and replays any signals
// buffered since construction.
func (f *TerminationForwarder) Attach(proc *os.Process) {
	f.procCh <- proc
}

// Stop uninstalls the handler and cancels any pending SIGKILL escalation.
func (f *TerminationForwarder) Stop() {
	close(f.done)
	signal.Stop(f.sigCh)
}
