package dist

import (
	"fmt"
	"os"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
)

// Source reads populate the reusable metadata cache. Keep every test's cache
// out of the developer's home, including tests that only resolve metadata.
func TestMain(m *testing.M) {
	directory, err := os.MkdirTemp("", "cjv-dist-tests-*")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := os.Setenv(config.EnvHome, directory); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := m.Run()
	_ = os.RemoveAll(directory)
	os.Exit(code)
}
