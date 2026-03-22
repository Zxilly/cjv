package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for updateSettingsAfterUninstall -- when a toolchain is removed,
// settings must be cleaned up: overrides referencing it removed, and
// a new default selected if needed.

func TestUpdateSettingsAfterUninstall_RemovesOverrides(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	// Create settings with overrides, some pointing to the uninstalled toolchain
	settings := config.DefaultSettings()
	settings.Overrides["C:\\project-a"] = "lts-1.0.5"
	settings.Overrides["C:\\project-b"] = "sts-2.0.0"
	settings.Overrides["C:\\project-c"] = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	// The uninstalled toolchain's overrides should be removed
	require.NoError(t, updateSettingsAfterUninstall("lts-1.0.5"))

	// Reload and verify
	reloaded, err := config.LoadSettings(filepath.Join(home, "settings.toml"))
	require.NoError(t, err)
	assert.NotContains(t, reloaded.Overrides, "C:\\project-a")
	assert.Contains(t, reloaded.Overrides, "C:\\project-b")
	assert.NotContains(t, reloaded.Overrides, "C:\\project-c")
}

func TestUpdateSettingsAfterUninstall_SelectsNewDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	// Create a remaining toolchain
	tcDir := filepath.Join(home, "toolchains")
	require.NoError(t, os.MkdirAll(filepath.Join(tcDir, "sts-2.0.0"), 0o755))

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	require.NoError(t, updateSettingsAfterUninstall("lts-1.0.5"))

	reloaded, _ := config.LoadSettings(filepath.Join(home, "settings.toml"))
	assert.Equal(t, "sts-2.0.0", reloaded.DefaultToolchain,
		"should pick remaining toolchain as new default")
}

func TestUpdateSettingsAfterUninstall_ClearsDefaultWhenNoneRemain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	// No other toolchains installed
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains"), 0o755))

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	require.NoError(t, updateSettingsAfterUninstall("lts-1.0.5"))

	reloaded, _ := config.LoadSettings(filepath.Join(home, "settings.toml"))
	assert.Empty(t, reloaded.DefaultToolchain,
		"should clear default when no toolchains remain")
}

func TestUpdateSettingsAfterUninstall_NoChangeWhenUnrelated(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "sts-2.0.0"
	settings.Overrides["C:\\dir"] = "sts-2.0.0"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	// Uninstalling a different toolchain should change nothing
	require.NoError(t, updateSettingsAfterUninstall("lts-1.0.5"))

	reloaded, _ := config.LoadSettings(filepath.Join(home, "settings.toml"))
	assert.Equal(t, "sts-2.0.0", reloaded.DefaultToolchain)
	assert.Equal(t, "sts-2.0.0", reloaded.Overrides["C:\\dir"])
}

