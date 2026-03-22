package proxy

import (
	"os"
	"os/signal"
)

// InterceptInterrupts keeps the parent process alive long enough to collect the
// child exit status while still allowing the child to receive Ctrl+C normally.
func InterceptInterrupts() func() {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt)

	done := make(chan struct{})
	go func() {
		for {
			select {
			case <-sigCh:
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
