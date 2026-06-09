//go:build windows

package proxy

import "os"

// terminationSignals on Windows is limited to Ctrl+C (os.Interrupt): the
// console delivers it to the whole process group including the child, and
// os.Process.Signal cannot relay arbitrary signals on Windows. The parent only
// needs to avoid exiting before the child does.
var terminationSignals = []os.Signal{os.Interrupt}

func forwardSignal(_ *os.Process, _ os.Signal, _ <-chan struct{}) {}
