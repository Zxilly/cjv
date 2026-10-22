package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExplicitConfigPathDoesNotDependOnDataHome(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv(EnvConfigFile, filepath.Join("ci", "settings.toml"))
	t.Setenv(EnvHome, filepath.Join(dir, "data"))
	path, err := SettingsPath()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, "ci", "settings.toml"), path)
	sf, err := DefaultSettingsFile()
	require.NoError(t, err)
	_, err = sf.Load()
	require.NoError(t, err, "a selected file does not need to exist")
	assert.NoFileExists(t, path)
	name := "lts-1.0.5"
	_, err = sf.Update(SettingsUpdate{DefaultToolchain: &name})
	require.NoError(t, err)
	assert.FileExists(t, path)
	assert.NoDirExists(t, filepath.Join(dir, "data"))
}

func TestAutoInstallEnvironmentDoesNotPersist(t *testing.T) {
	for _, value := range []string{"0", "1"} {
		t.Run(value, func(t *testing.T) {
			t.Setenv(EnvAutoInstall, value)
			sf := NewSettingsFile(filepath.Join(t.TempDir(), "settings.toml"))
			settings, err := sf.Load()
			require.NoError(t, err)
			enabled, err := ResolveAutoInstall(settings.AutoInstall)
			require.NoError(t, err)
			assert.Equal(t, value == "1", enabled)
			settings.DefaultToolchain = "lts"
			require.NoError(t, sf.Save(settings))
			data, err := os.ReadFile(sf.Path())
			require.NoError(t, err)
			assert.NotContains(t, string(data), "auto_install")
		})
	}
	t.Setenv(EnvAutoInstall, "sometimes")
	_, err := ResolveAutoInstall(true)
	require.ErrorContains(t, err, EnvAutoInstall)
}

func TestSettingsMutationWithoutHomeEnvironment(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(EnvHome, "")
	t.Setenv(EnvConfigFile, filepath.Join(dir, "settings.toml"))
	sf, err := DefaultSettingsFile()
	require.NoError(t, err)
	_, undo, err := sf.MutateWithUndo(func(s *Settings) (SettingsUpdate, error) {
		name := "lts"
		return SettingsUpdate{DefaultToolchain: &name}, nil
	})
	require.NoError(t, err)
	require.NoError(t, undo())
}
