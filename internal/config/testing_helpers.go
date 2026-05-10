package config

import "testing"

// IsolateForTest points CJV_HOME at tmpDir and redirects the resolved user
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
func IsolateForTest(t *testing.T, tmpDir string) {
	t.Helper()
	t.Setenv(EnvHome, tmpDir)
	SetUserHomeDirOverrideForTest(tmpDir)
	ResetDefaultSettingsFileCache()
	t.Cleanup(func() {
		SetUserHomeDirOverrideForTest("")
		ResetDefaultSettingsFileCache()
	})
}
