//go:build !windows

package utils

import (
	"fmt"
	"os/exec"
	"strings"
)

// ProcessName returns the name of the process with the given PID.
func ProcessName(pid int) (string, error) {
	out, err := exec.Command("ps", "-p", fmt.Sprintf("%d", pid), "-o", "comm=").Output()
	if err != nil {
		return "", err
	}
	name := strings.TrimSpace(string(out))
	if name == "" {
		return "", fmt.Errorf("empty process name for pid %d", pid)
	}
	return name, nil
}
