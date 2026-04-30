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
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.5"), 0o755))
	settings := config.DefaultSettings()
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	cmd := &cobra.Command{}
	err := runDefault(cmd, []string{"lts-1.0.5"})
	require.NoError(t, err)

	reloaded, _ := config.LoadSettings(filepath.Join(home, "settings.toml"))
	assert.Equal(t, "lts-1.0.5", reloaded.DefaultToolchain)
}

func TestRunDefault_RejectsTargetVariant(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	name := "sts-2.0.0-win32-x64-ohos"
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", name), 0o755))
	settings := config.DefaultSettings()
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	cmd := &cobra.Command{}
	err := runDefault(cmd, []string{name})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "target variant")

	reloaded, err := config.LoadSettings(filepath.Join(home, "settings.toml"))
	require.NoError(t, err)
	assert.Empty(t, reloaded.DefaultToolchain)
}

func TestRunDefault_ClearsDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	cmd := &cobra.Command{}
	err := runDefault(cmd, []string{"none"})
	require.NoError(t, err)

	reloaded, _ := config.LoadSettings(filepath.Join(home, "settings.toml"))
	assert.Empty(t, reloaded.DefaultToolchain)
}

func TestRunDefault_ShowsCurrentDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	cmd := &cobra.Command{}
	err := runDefault(cmd, nil)
	assert.NoError(t, err)
}

func TestShowDefault_WithDefaultSet(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	// Should succeed without error
	err := showDefault()
	assert.NoError(t, err)
}

func TestShowDefault_NoDefaultSet(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	settings := config.DefaultSettings()
	settings.DefaultToolchain = ""
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	// Should succeed (prints "no default" message, not an error)
	err := showDefault()
	assert.NoError(t, err)
}

func TestShowDefault_NoSettingsFile(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	// No settings file — LoadSettings returns defaults
	err := showDefault()
	assert.NoError(t, err)
}
