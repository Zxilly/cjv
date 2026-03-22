//go:build !windows

package proxy

import (
	"context"
	"syscall"
)

// execTool replaces the current process via syscall.Exec.
// The context parameter is accepted for API consistency but is unused
// because syscall.Exec replaces the entire process image.
func execTool(_ context.Context, binary string, args []string, env []string) error {
	return syscall.Exec(binary, append([]string{binary}, args...), env)
}
