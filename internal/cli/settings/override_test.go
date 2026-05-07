package settings

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOverrideSetPreservesBareVersion(t *testing.T) {
	home := t.TempDir()
	dir := t.TempDir()
	t.Setenv(config.EnvHome, home)

	prev := overrideSetPath
	overrideSetPath = dir
	defer func() {
		overrideSetPath = prev
	}()

	require.NoError(t, overrideSetCmd.RunE(overrideSetCmd, []string{"1.0.5"}))

	settings, err := config.LoadSettings(filepath.Join(home, "settings.toml"))
	require.NoError(t, err)
	assert.Equal(t, "1.0.5", settings.Overrides[config.NormalizePath(dir)])
}

func TestOverrideSetRejectsTargetVariant(t *testing.T) {
	home := t.TempDir()
	dir := t.TempDir()
	t.Setenv(config.EnvHome, home)

	prev := overrideSetPath
	overrideSetPath = dir
	defer func() {
		overrideSetPath = prev
	}()

	err := overrideSetCmd.RunE(overrideSetCmd, []string{"sts-2.0.0-win32-x64-ohos"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "target variant")

	settings, err := config.LoadSettings(filepath.Join(home, "settings.toml"))
	require.NoError(t, err)
	assert.Empty(t, settings.Overrides)
}

func TestResolveOverrideDir_WithExplicitPath(t *testing.T) {
	dir := t.TempDir()
	resolved, err := resolveOverrideDir(dir)
	require.NoError(t, err)
	assert.NotEmpty(t, resolved)
}

func TestResolveOverrideDir_FallsBackToCwd(t *testing.T) {
	cwd := t.TempDir()
	origCwd, _ := os.Getwd()
	os.Chdir(cwd)
	defer os.Chdir(origCwd)

	resolved, err := resolveOverrideDir("")
	require.NoError(t, err)
	assert.NotEmpty(t, resolved)
}

// Tests for unsetNonexistentOverrides -- cleans up overrides pointing
// to directories that no longer exist on disk.

func TestUnsetNonexistentOverrides_RemovesStale(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	existingDir := t.TempDir() // this dir exists
	goneDir := filepath.Join(t.TempDir(), "gone")
	os.RemoveAll(goneDir) // ensure it doesn't exist

	stg := config.DefaultSettings()
	stg.Overrides[existingDir] = "lts-1.0.5"
	stg.Overrides[goneDir] = "sts-2.0.0"
	settingsPath := filepath.Join(home, "settings.toml")
	require.NoError(t, config.SaveSettings(&stg, settingsPath))

	sf := config.NewSettingsFile(settingsPath)
	require.NoError(t, unsetNonexistentOverrides(&stg, sf))

	assert.Contains(t, stg.Overrides, existingDir,
		"override for existing dir should be preserved")
	assert.NotContains(t, stg.Overrides, goneDir,
		"override for gone dir should be removed")
}

func TestUnsetNonexistentOverrides_NoOpWhenAllExist(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	dir1 := t.TempDir()
	dir2 := t.TempDir()

	stg := config.DefaultSettings()
	stg.Overrides[dir1] = "lts-1.0.5"
	stg.Overrides[dir2] = "sts-2.0.0"
	settingsPath := filepath.Join(home, "settings.toml")
	require.NoError(t, config.SaveSettings(&stg, settingsPath))

	sf := config.NewSettingsFile(settingsPath)
	require.NoError(t, unsetNonexistentOverrides(&stg, sf))

	assert.Len(t, stg.Overrides, 2, "all overrides should be preserved")
}

func TestOverrideUnsetCommandRemovesMatchingNormalizedPath(t *testing.T) {
	home := t.TempDir()
	dir := t.TempDir()
	t.Setenv(config.EnvHome, home)
	config.ResetDefaultSettingsFileCache()
	t.Cleanup(config.ResetDefaultSettingsFileCache)

	settings := config.DefaultSettings()
	settings.Overrides[config.NormalizePath(dir)] = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	oldPath := overrideUnsetPath
	oldNonexistent := overrideUnsetNonexistent
	overrideUnsetPath = dir
	overrideUnsetNonexistent = false
	t.Cleanup(func() {
		overrideUnsetPath = oldPath
		overrideUnsetNonexistent = oldNonexistent
	})

	require.NoError(t, overrideUnsetCmd.RunE(overrideUnsetCmd, nil))

	got, err := config.LoadSettings(filepath.Join(home, "settings.toml"))
	require.NoError(t, err)
	assert.Empty(t, got.Overrides)
}

func TestOverrideUnsetCommandErrorsWhenMissing(t *testing.T) {
	home := t.TempDir()
	dir := t.TempDir()
	t.Setenv(config.EnvHome, home)
	config.ResetDefaultSettingsFileCache()
	t.Cleanup(config.ResetDefaultSettingsFileCache)

	oldPath := overrideUnsetPath
	oldNonexistent := overrideUnsetNonexistent
	overrideUnsetPath = dir
	overrideUnsetNonexistent = false
	t.Cleanup(func() {
		overrideUnsetPath = oldPath
		overrideUnsetNonexistent = oldNonexistent
	})

	err := overrideUnsetCmd.RunE(overrideUnsetCmd, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no override")
}

func TestOverrideListCommandHandlesEmptyAndSortedEntries(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)
	config.ResetDefaultSettingsFileCache()
	t.Cleanup(config.ResetDefaultSettingsFileCache)

	require.NoError(t, overrideListCmd.RunE(overrideListCmd, nil))

	settings := config.DefaultSettings()
	settings.Overrides[filepath.Join(home, "b")] = "sts"
	settings.Overrides[filepath.Join(home, "a")] = "lts"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	require.NoError(t, overrideListCmd.RunE(overrideListCmd, nil))
}
