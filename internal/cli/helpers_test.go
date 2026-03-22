package cli

import (
	"os"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
)

// TestMain configures the test environment for the cli package.
// NOTE: cli tests must NOT use t.Parallel() because cobra stores flag values
// in package-level variables (e.g. forceInstall, noSelfUpdate, overrideSetPath)
// that are mutated during command execution. Parallel test runs would race on
// this shared state.
func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

func runTests(m *testing.M) int {
	if os.Getenv("CI") == "true" {
		// CI mode: let ensurePathConfigured run for real so the actual
		// code path is exercised, but wrap the run in a platform-specific
		// guard that saves and restores the system PATH afterward
		// (saves and restores the system PATH after the test run).
		return runWithPathGuard(m)
	}

	// Local dev: disable PATH writes entirely to avoid polluting the
	// developer's system. Each test uses a unique t.TempDir() as
	// CJV_HOME, so without this override every test run would append
	// a new temp-dir entry to the system PATH.
	ensurePathConfiguredFn = func() {}
	config.ResetDefaultSettingsFileCache()
	return m.Run()
}
