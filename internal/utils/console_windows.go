//go:build windows

package utils

import (
	"bufio"
	"fmt"
	"os"
	"unsafe"

	"golang.org/x/sys/windows"
)

var (
	kernel32                  = windows.NewLazySystemDLL("kernel32.dll")
	procGetConsoleProcessList = kernel32.NewProc("GetConsoleProcessList")
	procGetConsoleOutputCP    = kernel32.NewProc("GetConsoleOutputCP")
	procSetConsoleOutputCP    = kernel32.NewProc("SetConsoleOutputCP")
)

const cpUTF8 = 65001

// EnableConsoleUTF8 switches the console's output code page to UTF-8 so that
// cjv's UTF-8 output (Chinese messages, paths containing a Chinese username)
// renders correctly. Legacy conhost otherwise interprets output bytes in the
// active OEM code page (e.g. CP936 on a Chinese system) and turns multi-byte
// UTF-8 sequences into mojibake.
//
// It returns a function that restores the previous code page; callers should
// defer it so the hosting shell is left as it was found. The returned function
// is always safe to call: when no console is attached (e.g. output redirected
// to a pipe) or the page is already UTF-8 (e.g. Windows Terminal), this is a
// no-op and nothing is changed.
func EnableConsoleUTF8() func() {
	prev, _, _ := procGetConsoleOutputCP.Call()
	// GetConsoleOutputCP returns 0 on failure, which includes the case of no
	// console being attached. Nothing to switch or restore then.
	if prev == 0 || prev == cpUTF8 {
		return func() {}
	}
	if ret, _, _ := procSetConsoleOutputCP.Call(uintptr(cpUTF8)); ret == 0 {
		return func() {} // failed to switch; leave the console untouched
	}
	return func() {
		_, _, _ = procSetConsoleOutputCP.Call(prev)
	}
}

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
