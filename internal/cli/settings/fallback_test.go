package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettingChangePreservesFallback(t *testing.T) {
	dir := t.TempDir()
	config.IsolateForTest(t, dir)
	fallback := filepath.Join(dir, "fallback.toml")
	t.Setenv(config.EnvFallbackSettings, fallback)
	require.NoError(t, os.WriteFile(fallback, []byte(`dist_server = "https://old.example/cjv"
default_toolchain = "lts-1.0.5"
auto_self_update = "disable"
auto_install = false
`), 0o600))

	require.NoError(t, executeSettings(t, "set", "auto-install", "true"))
	path, err := config.SettingsPath()
	require.NoError(t, err)
	var stored map[string]any
	_, err = toml.DecodeFile(path, &stored)
	require.NoError(t, err)
	assert.Equal(t, true, stored["auto_install"])
	for _, key := range []string{"dist_server", "default_toolchain", "auto_self_update", "manifest_url"} {
		assert.NotContains(t, stored, key)
	}

	require.NoError(t, os.WriteFile(fallback, []byte(`dist_server = "https://new.example/cjv"
default_toolchain = "lts-1.0.6"
auto_self_update = "check"
auto_install = false
`), 0o600))
	config.ResetDefaultSettingsFileCache()
	_, loaded, err := config.LoadDefaultSettings()
	require.NoError(t, err)
	assert.Equal(t, "https://new.example/cjv", loaded.DistServer)
	assert.Equal(t, "lts-1.0.6", loaded.DefaultToolchain)
	assert.Equal(t, "check", loaded.AutoSelfUpdate)
	assert.True(t, loaded.AutoInstall)
}

func TestSetPinsValueEqualToFallback(t *testing.T) {
	dir := t.TempDir()
	config.IsolateForTest(t, dir)
	fallback := filepath.Join(dir, "fallback.toml")
	t.Setenv(config.EnvFallbackSettings, fallback)
	require.NoError(t, os.WriteFile(fallback, []byte("auto_install = false\n"), 0o600))

	require.NoError(t, executeSettings(t, "set", "auto-install", "false"))
	require.NoError(t, os.WriteFile(fallback, []byte("auto_install = true\n"), 0o600))
	config.ResetDefaultSettingsFileCache()
	_, loaded, err := config.LoadDefaultSettings()
	require.NoError(t, err)
	assert.False(t, loaded.AutoInstall)
}

func TestSetEmptyHomeSuppressesFallback(t *testing.T) {
	dir := t.TempDir()
	config.IsolateForTest(t, dir)
	t.Setenv(config.EnvHome, "")
	fallback := filepath.Join(dir, "fallback.toml")
	t.Setenv(config.EnvFallbackSettings, fallback)
	require.NoError(t, os.WriteFile(fallback, []byte("home = '"+filepath.Join(dir, "managed")+"'\n"), 0o600))

	require.NoError(t, executeSettings(t, "set", "home", ""))
	config.ResetDefaultSettingsFileCache()
	home, err := config.Home()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(dir, ".cjv"), home)
}

func TestDefaultAndOverrideKeepUnrelatedFallback(t *testing.T) {
	dir := t.TempDir()
	config.IsolateForTest(t, dir)
	fallback := filepath.Join(dir, "fallback.toml")
	t.Setenv(config.EnvFallbackSettings, fallback)
	require.NoError(t, os.WriteFile(fallback, []byte("default_toolchain = 'lts-1.0.5'\nauto_install = false\n"), 0o600))

	require.NoError(t, executeSettings(t, "default", "none"))
	require.NoError(t, executeSettings(t, "override", "set", "sts", "--path", dir))
	require.NoError(t, executeSettings(t, "override", "unset", "--path", dir))
	require.NoError(t, os.WriteFile(fallback, []byte("default_toolchain = 'lts-1.0.6'\nauto_install = true\n"), 0o600))
	config.ResetDefaultSettingsFileCache()
	_, loaded, err := config.LoadDefaultSettings()
	require.NoError(t, err)
	assert.Empty(t, loaded.DefaultToolchain, "explicit none suppresses the system default")
	assert.Empty(t, loaded.Overrides)
	assert.True(t, loaded.AutoInstall, "unrelated system settings remain inherited")
}
