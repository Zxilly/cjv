package settings

import (
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSetHomeCommandPersists verifies that `cjv settings set home <path>`
// writes the absolute path into ~/.cjv/settings.toml and that subsequent
// Home() resolution picks it up when CJV_HOME is unset.
func TestSetHomeCommandPersists(t *testing.T) {
	tmp := t.TempDir()
	config.IsolateForTest(t, tmp)
	t.Setenv(config.EnvHome, "") // ensure persisted source wins

	target := filepath.Join(tmp, "data")
	require.NoError(t, setHomeCmd.RunE(setHomeCmd, []string{target}))

	// settings.toml should now contain the absolute path.
	settingsPath, err := config.SettingsPath()
	require.NoError(t, err)
	settings, err := config.LoadSettings(settingsPath)
	require.NoError(t, err)
	expectedAbs, _ := filepath.Abs(target)
	assert.Equal(t, expectedAbs, settings.Home)

	// Home() should resolve to the persisted value with HomeSourcePersisted.
	config.ResetDefaultSettingsFileCache()
	got, src, err := config.ResolveHomeWithSource()
	require.NoError(t, err)
	assert.Equal(t, expectedAbs, got)
	assert.Equal(t, config.HomeSourcePersisted, src)
}

// TestSetHomeCommandEnvStillWins verifies that even after persisting,
// CJV_HOME continues to override.
func TestSetHomeCommandEnvStillWins(t *testing.T) {
	tmp := t.TempDir()
	config.IsolateForTest(t, tmp)
	t.Setenv(config.EnvHome, "") // first persist without env

	persisted := filepath.Join(tmp, "persisted")
	require.NoError(t, setHomeCmd.RunE(setHomeCmd, []string{persisted}))
	config.ResetDefaultSettingsFileCache()

	// Now set CJV_HOME to something else.
	envHome := filepath.Join(tmp, "env-home")
	t.Setenv(config.EnvHome, envHome)

	got, src, err := config.ResolveHomeWithSource()
	require.NoError(t, err)
	expectedAbs, _ := filepath.Abs(envHome)
	assert.Equal(t, expectedAbs, got)
	assert.Equal(t, config.HomeSourceEnv, src)
}

// TestSetHomeCommandEmptyClears verifies passing "" clears the persisted home.
func TestSetHomeCommandEmptyClears(t *testing.T) {
	tmp := t.TempDir()
	config.IsolateForTest(t, tmp)
	t.Setenv(config.EnvHome, "")

	// First persist a value.
	require.NoError(t, setHomeCmd.RunE(setHomeCmd, []string{filepath.Join(tmp, "data")}))
	// Then clear.
	require.NoError(t, setHomeCmd.RunE(setHomeCmd, []string{""}))
	config.ResetDefaultSettingsFileCache()

	settingsPath, err := config.SettingsPath()
	require.NoError(t, err)
	settings, err := config.LoadSettings(settingsPath)
	require.NoError(t, err)
	assert.Equal(t, "", settings.Home)

	// Home() should now fall back to default.
	got, src, err := config.ResolveHomeWithSource()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tmp, ".cjv"), got)
	assert.Equal(t, config.HomeSourceDefault, src)
}
