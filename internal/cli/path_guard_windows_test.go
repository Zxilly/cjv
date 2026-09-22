//go:build windows

package cli

import (
	"fmt"
	"os"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/testutil"
)

func runWithPathGuard(m *testing.M) int {
	guard, err := testutil.SaveRegistryPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not save registry PATH, disabling PATH writes: %v\n", err)
		if setErr := os.Setenv(config.EnvNoPathSetup, "1"); setErr != nil {
			panic(setErr)
		}
		return m.Run()
	}
	defer guard.Restore()

	// Each application's default PATH adapter remains enabled in guarded CI.
	return m.Run()
}
