//go:build !windows

package cli

import "testing"

func runWithPathGuard(m *testing.M) int {
	// On Unix CI, ensurePathConfigured writes to shell rc files
	// (~/.bashrc, ~/.zshrc). CI containers are ephemeral so these
	// modifications are harmless and get discarded with the container.
	// Letting the code run exercises the real code path in CI.
	return m.Run()
}
