//go:build integration && !windows

package integration

import "testing"

func runWithRegistryGuard(m *testing.M) int {
	// On Unix CI, containers are ephemeral — no need to guard anything.
	return m.Run()
}
