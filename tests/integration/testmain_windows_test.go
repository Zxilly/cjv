//go:build integration && windows

package integration

import (
	"fmt"
	"os"
	"testing"

	"github.com/Zxilly/cjv/internal/testutil"
)

func runWithRegistryGuard(m *testing.M) int {
	guard, err := testutil.SaveRegistryPath()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not save registry PATH: %v\n", err)
		return m.Run()
	}
	defer guard.Restore()
	return m.Run()
}
