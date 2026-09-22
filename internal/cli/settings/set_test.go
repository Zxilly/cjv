package settings

import (
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetCommandsUpdateSettings(t *testing.T) {
	tmp := t.TempDir()
	config.IsolateForTest(t, tmp)

	require.NoError(t, executeSettings(t, "set", "auto-self-update", config.AutoSelfUpdateDisable))
	require.NoError(t, executeSettings(t, "set", "auto-install", "false"))
	require.NoError(t, executeSettings(t, "set", "default-host", "linux-amd64"))

	path, err := config.SettingsPath()
	require.NoError(t, err)
	settings, err := config.LoadSettings(path)
	require.NoError(t, err)
	assert.Equal(t, config.AutoSelfUpdateDisable, settings.AutoSelfUpdate)
	assert.False(t, settings.AutoInstall)
	assert.Equal(t, "linux-amd64", settings.DefaultHost)
}

func TestSetCommandsRejectInvalidValues(t *testing.T) {
	tmp := t.TempDir()
	config.IsolateForTest(t, tmp)

	require.Error(t, executeSettings(t, "set", "auto-self-update", "sometimes"))
	require.Error(t, executeSettings(t, "set", "auto-install", "maybe"))
	require.Error(t, executeSettings(t, "set", "default-host", "plan9-amd64"))
}
