package config

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	ResetDefaultSettingsFileCache()
	os.Exit(m.Run())
}

// isolateForTest mirrors configtest.Isolate for this package's own tests,
// which cannot import configtest without an import cycle.
func isolateForTest(t *testing.T, tmpDir string) {
	t.Helper()
	t.Setenv(EnvHome, tmpDir)
	SetUserHomeDirOverrideForTest(tmpDir)
	ResetDefaultSettingsFileCache()
	t.Cleanup(func() {
		SetUserHomeDirOverrideForTest("")
		ResetDefaultSettingsFileCache()
	})
}
