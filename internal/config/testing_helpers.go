package config

import (
	"runtime"
	"testing"
)

// IsolateForTest points both CJV_HOME and the OS user home directory at the
// given tmpDir, then resets the package's caches so subsequent calls to
// Home(), SettingsPath() and DefaultSettingsFile() observe the new layout.
//
// Use this in tests that touch the persisted settings file (which now lives
// at <user-home>/.cjv/settings.toml regardless of CJV_HOME), so the test
// cannot accidentally read or write the developer's real ~/.cjv.
//
// The helper registers a t.Cleanup that resets the caches again, ensuring
// later tests in the same binary do not observe stale cached values.
func IsolateForTest(t *testing.T, tmpDir string) {
	t.Helper()
	t.Setenv(EnvHome, tmpDir)
	switch runtime.GOOS {
	case "windows":
		t.Setenv("USERPROFILE", tmpDir)
	case "plan9":
		t.Setenv("home", tmpDir)
	default:
		t.Setenv("HOME", tmpDir)
	}
	ResetCachedUserHomeDir()
	ResetDefaultSettingsFileCache()
	t.Cleanup(func() {
		ResetCachedUserHomeDir()
		ResetDefaultSettingsFileCache()
	})
}
