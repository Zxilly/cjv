//go:build windows

package utils

import (
	"bufio"
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

var procGetConsoleProcessList = windows.NewLazySystemDLL("kernel32.dll").NewProc("GetConsoleProcessList")

// PauseIfStandaloneConsole blocks for an Enter key when this process is the
// sole owner of its console — i.e. the conhost window opened just to host us
// and will close on exit. Keeps a double-clicked installer's output visible
// until the user dismisses it.
func PauseIfStandaloneConsole() {
	if !ownsConsole() {
		return
	}
	fmt.Fprintln(os.Stderr)
	fmt.Fprint(os.Stderr, "按 Enter 键退出...")
	_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
}

func ownsConsole() bool {
	var buf [2]uint32
	r, _, _ := procGetConsoleProcessList.Call(uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	return r == 1
}
