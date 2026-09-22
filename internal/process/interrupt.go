package process

import (
	"os"
	"os/signal"
)

// terminationForwarder keeps the parent alive long enough to collect the
// child's exit status and relays termination signals to the child.
// Interception starts before the child is spawned, so signals arriving before
// attach are buffered and replayed once the process handle is available.
// Ctrl+C is not forwarded: the terminal already delivers it to the child.
type terminationForwarder struct {
	sigCh  chan os.Signal
	procCh chan *os.Process
	done   chan struct{}
}

func newTerminationForwarder() *terminationForwarder {
	f := &terminationForwarder{
		sigCh:  make(chan os.Signal, 1),
		procCh: make(chan *os.Process, 1),
		done:   make(chan struct{}),
	}
	signal.Notify(f.sigCh, terminationSignals...)
	go f.run()
	return f
}

func (f *terminationForwarder) run() {
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

func (f *terminationForwarder) attach(proc *os.Process) {
	f.procCh <- proc
}

func (f *terminationForwarder) stop() {
	close(f.done)
	signal.Stop(f.sigCh)
}
