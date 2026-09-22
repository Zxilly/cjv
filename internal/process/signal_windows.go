//go:build windows

package process

import "os"

// The console delivers Ctrl+C to the whole process group, including the
// child. The parent only needs to avoid exiting before the child does;
// os.Process.Signal cannot relay arbitrary signals on Windows.
var terminationSignals = []os.Signal{os.Interrupt}

func forwardSignal(_ *os.Process, _ os.Signal, _ <-chan struct{}) {}
