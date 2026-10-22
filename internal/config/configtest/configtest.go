// Package configtest provides test helpers for code that reads cjv settings
// and data layout. It lives outside package config so production builds do
// not depend on the testing package.
package configtest

import (
	"testing"

	"github.com/Zxilly/cjv/internal/config"
)

// Isolate points CJV_HOME at tmpDir and redirects the resolved user
// home dir (used by SettingsPath() and the default-home branch of Home())
// to the same tmpDir, then resets the package's caches so subsequent calls
// observe the new layout.
//
// Use this in tests that touch the persisted settings file (which lives at
// <user-home>/.cjv/settings.toml regardless of CJV_HOME), so the test
// cannot accidentally read or write the developer's real ~/.cjv.
//
// The OS HOME/USERPROFILE env vars are intentionally NOT modified: doing so
// leaks into subprocesses the test may launch — e.g. `go version` writes
// telemetry into <tmpDir>/Library/Application Support/go/..., which races
// with t.TempDir's RemoveAll cleanup and produces sporadic "directory not
// empty" failures on macOS CI. The package-internal override avoids that.
//
// The helper registers a t.Cleanup that clears the override and resets the
// caches, so later tests in the same binary do not observe stale state.
func Isolate(t *testing.T, tmpDir string) {
	t.Helper()
	t.Setenv(config.EnvHome, tmpDir)
	t.Setenv(config.EnvConfigFile, "")
	t.Setenv(config.EnvAutoInstall, "")
	config.SetUserHomeDirOverrideForTest(tmpDir)
	config.ResetDefaultSettingsFileCache()
	t.Cleanup(func() {
		config.SetUserHomeDirOverrideForTest("")
		config.ResetDefaultSettingsFileCache()
	})
}
