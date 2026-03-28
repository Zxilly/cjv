//go:build windows

package utils

import (
	"fmt"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

// ProcessName returns the name of the process with the given PID.
func ProcessName(pid int) (string, error) {
	h, err := windows.CreateToolhelp32Snapshot(windows.TH32CS_SNAPPROCESS, 0)
	if err != nil {
		return "", err
	}
	defer func() { _ = windows.CloseHandle(h) }()

	var pe windows.ProcessEntry32
	pe.Size = uint32(unsafe.Sizeof(pe))
	if err := windows.Process32First(h, &pe); err != nil {
		return "", err
	}
	for {
		if pe.ProcessID == uint32(pid) {
			return syscall.UTF16ToString(pe.ExeFile[:]), nil
		}
		if err := windows.Process32Next(h, &pe); err != nil {
			break
		}
	}
	return "", fmt.Errorf("process %d not found", pid)
}
