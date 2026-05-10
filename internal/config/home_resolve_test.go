package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestResolveHome_EnvWinsOverPersisted verifies CJV_HOME beats settings.home.
func TestResolveHome_EnvWinsOverPersisted(t *testing.T) {
	tmp := t.TempDir()
	IsolateForTest(t, tmp)

	// Persist a different home in settings.toml.
	persisted := filepath.Join(tmp, "persisted-home")
	s := DefaultSettings()
	s.Home = persisted
	settingsDir := filepath.Join(tmp, ".cjv")
	require.NoError(t, os.MkdirAll(settingsDir, 0o755))
	require.NoError(t, SaveSettings(&s, filepath.Join(settingsDir, "settings.toml")))
	ResetDefaultSettingsFileCache()

	// CJV_HOME (set by IsolateForTest to tmp) wins.
	got, src, err := ResolveHomeWithSource()
	require.NoError(t, err)
	expected, _ := filepath.Abs(tmp)
	assert.Equal(t, expected, got)
	assert.Equal(t, HomeSourceEnv, src)
}

// TestResolveHome_PersistedWinsOverDefault verifies settings.home is used when CJV_HOME is unset.
func TestResolveHome_PersistedWinsOverDefault(t *testing.T) {
	tmp := t.TempDir()
	IsolateForTest(t, tmp)
	t.Setenv(EnvHome, "") // override IsolateForTest

	persisted := filepath.Join(tmp, "persisted-home")
	s := DefaultSettings()
	s.Home = persisted
	settingsDir := filepath.Join(tmp, ".cjv")
	require.NoError(t, os.MkdirAll(settingsDir, 0o755))
	require.NoError(t, SaveSettings(&s, filepath.Join(settingsDir, "settings.toml")))
	ResetDefaultSettingsFileCache()

	got, src, err := ResolveHomeWithSource()
	require.NoError(t, err)
	expected, _ := filepath.Abs(persisted)
	assert.Equal(t, expected, got)
	assert.Equal(t, HomeSourcePersisted, src)
}

// TestResolveHome_DefaultWhenNothingSet verifies the fallback to <user-home>/.cjv.
func TestResolveHome_DefaultWhenNothingSet(t *testing.T) {
	tmp := t.TempDir()
	IsolateForTest(t, tmp)
	t.Setenv(EnvHome, "")
	// No settings.toml exists.

	got, src, err := ResolveHomeWithSource()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmp, ".cjv"), got)
	assert.Equal(t, HomeSourceDefault, src)
}

// TestSettingsPath_IgnoresCJVHome verifies SettingsPath() always lives under the
// OS user home, regardless of CJV_HOME.
func TestSettingsPath_IgnoresCJVHome(t *testing.T) {
	tmp := t.TempDir()
	IsolateForTest(t, tmp)

	// Override CJV_HOME to a different value than the user home, then check
	// that SettingsPath() ignores it.
	other := t.TempDir()
	t.Setenv(EnvHome, other)

	got, err := SettingsPath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmp, ".cjv", "settings.toml"), got)
}

// TestHomeSourceString smoke-tests the String method.
func TestHomeSourceString(t *testing.T) {
	assert.Equal(t, "env", HomeSourceEnv.String())
	assert.Equal(t, "persisted", HomeSourcePersisted.String())
	assert.Equal(t, "default", HomeSourceDefault.String())
}
