package cli

import (
	"io"
	"os"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/stretchr/testify/require"
)

// TestMain configures the test environment for the cli package.
// Tests isolate process environment and sometimes redirect os.Stdout, so they
// remain sequential even though each invocation owns its command state.
func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	if os.Getenv("CI") == "true" {
		// CI mode: let reachable.ConfigurePath run for real so the actual
		// code path is exercised, but wrap the run in a platform-specific
		// guard that saves and restores the system PATH afterward
		// (saves and restores the system PATH after the test run).
		return runWithPathGuard(m)
	}

	// Local dev: disable PATH writes entirely to avoid polluting the
	// developer's system. Each test uses a unique t.TempDir() as
	// CJV_HOME, so without this override every test run would append
	// a new temp-dir entry to the system PATH.
	if err := os.Setenv(config.EnvNoPathSetup, "1"); err != nil {
		panic(err)
	}
	config.ResetDefaultSettingsFileCache()
	return m.Run()
}

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	require.NoError(t, err)
	os.Stdout = w
	defer func() {
		os.Stdout = oldStdout
	}()

	runErr := fn()

	os.Stdout = oldStdout
	require.NoError(t, w.Close())
	data, readErr := io.ReadAll(r)
	require.NoError(t, readErr)
	require.NoError(t, r.Close())
	return string(data), runErr
}
