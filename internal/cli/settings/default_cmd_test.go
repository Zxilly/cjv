package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for runDefault — sets, shows, or clears the default toolchain.

func TestRunDefault_SetsDefault(t *testing.T) {
	tmp := t.TempDir()
	config.IsolateForTest(t, tmp)

	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "toolchains", "lts-1.0.5"), 0o755))
	settings := config.DefaultSettings()
	settingsPath, err := config.SettingsPath()
	require.NoError(t, err)
	require.NoError(t, config.SaveSettings(&settings, settingsPath))

	cmd := &cobra.Command{}
	err = runDefault(cmd, []string{"lts-1.0.5"})
	require.NoError(t, err)

	reloaded, _ := config.LoadSettings(settingsPath)
	assert.Equal(t, "lts-1.0.5", reloaded.DefaultToolchain)
}

func TestRunDefault_RejectsTargetVariant(t *testing.T) {
	tmp := t.TempDir()
	config.IsolateForTest(t, tmp)

	name := "sts-2.0.0-win32-x64-ohos"
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "toolchains", name), 0o755))
	settings := config.DefaultSettings()
	settingsPath, err := config.SettingsPath()
	require.NoError(t, err)
	require.NoError(t, config.SaveSettings(&settings, settingsPath))

	cmd := &cobra.Command{}
	err = runDefault(cmd, []string{name})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "target variant")

	reloaded, err := config.LoadSettings(settingsPath)
	require.NoError(t, err)
	assert.Empty(t, reloaded.DefaultToolchain)
}

func TestRunDefault_ClearsDefault(t *testing.T) {
	tmp := t.TempDir()
	config.IsolateForTest(t, tmp)

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	settingsPath, err := config.SettingsPath()
	require.NoError(t, err)
	require.NoError(t, config.SaveSettings(&settings, settingsPath))

	cmd := &cobra.Command{}
	err = runDefault(cmd, []string{"none"})
	require.NoError(t, err)

	reloaded, _ := config.LoadSettings(settingsPath)
	assert.Empty(t, reloaded.DefaultToolchain)
}

func TestRunDefault_ShowsCurrentDefault(t *testing.T) {
	tmp := t.TempDir()
	config.IsolateForTest(t, tmp)

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	settingsPath, err := config.SettingsPath()
	require.NoError(t, err)
	require.NoError(t, config.SaveSettings(&settings, settingsPath))

	cmd := &cobra.Command{}
	err = runDefault(cmd, nil)
	assert.NoError(t, err)
}

func TestShowDefault_WithDefaultSet(t *testing.T) {
	tmp := t.TempDir()
	config.IsolateForTest(t, tmp)

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	settingsPath, err := config.SettingsPath()
	require.NoError(t, err)
	require.NoError(t, config.SaveSettings(&settings, settingsPath))

	// Should succeed without error
	err = showDefault()
	assert.NoError(t, err)
}

func TestShowDefault_NoDefaultSet(t *testing.T) {
	tmp := t.TempDir()
	config.IsolateForTest(t, tmp)

	settings := config.DefaultSettings()
	settings.DefaultToolchain = ""
	settingsPath, err := config.SettingsPath()
	require.NoError(t, err)
	require.NoError(t, config.SaveSettings(&settings, settingsPath))

	// Should succeed (prints "no default" message, not an error)
	err = showDefault()
	assert.NoError(t, err)
}

func TestShowDefault_NoSettingsFile(t *testing.T) {
	tmp := t.TempDir()
	config.IsolateForTest(t, tmp)

	// No settings file — LoadSettings returns defaults
	err := showDefault()
	assert.NoError(t, err)
}
