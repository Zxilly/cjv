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

	require.NoError(t, setAutoSelfUpdateCmd.RunE(setAutoSelfUpdateCmd, []string{config.AutoSelfUpdateDisable}))
	require.NoError(t, setAutoInstallCmd.RunE(setAutoInstallCmd, []string{"false"}))
	require.NoError(t, setDefaultHostCmd.RunE(setDefaultHostCmd, []string{"linux-amd64"}))
	require.NoError(t, setGitCodeAPIKeyCmd.RunE(setGitCodeAPIKeyCmd, []string{"secret"}))

	path, err := config.SettingsPath()
	require.NoError(t, err)
	settings, err := config.LoadSettings(path)
	require.NoError(t, err)
	assert.Equal(t, config.AutoSelfUpdateDisable, settings.AutoSelfUpdate)
	assert.False(t, settings.AutoInstall)
	assert.Equal(t, "linux-amd64", settings.DefaultHost)
	assert.Equal(t, "secret", settings.GitCodeAPIKey)
}

func TestSetCommandsRejectInvalidValues(t *testing.T) {
	tmp := t.TempDir()
	config.IsolateForTest(t, tmp)

	require.Error(t, setAutoSelfUpdateCmd.RunE(setAutoSelfUpdateCmd, []string{"sometimes"}))
	require.Error(t, setAutoInstallCmd.RunE(setAutoInstallCmd, []string{"maybe"}))
	require.Error(t, setDefaultHostCmd.RunE(setDefaultHostCmd, []string{"plan9-amd64"}))
}
