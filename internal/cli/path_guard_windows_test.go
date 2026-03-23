//go:build windows

package cli

import (
	"fmt"
	"os"
	"testing"

	"github.com/Zxilly/cjv/internal/testutil"
)

func runWithPathGuard(m *testing.M) int {
	guard, err := testutil.SaveRegistryPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not save registry PATH, disabling PATH writes: %v\n", err)
		ensurePathConfiguredFn = func() {}
		return m.Run()
	}
	defer guard.Restore()

	// Let ensurePathConfiguredFn keep its default value (the real
	// ensurePathConfigured) so that CI exercises the actual code path.
	return m.Run()
}
