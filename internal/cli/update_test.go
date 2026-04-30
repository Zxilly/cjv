package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for update-related functions.

func TestUpdateAll_NoToolchains(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	_, _, err := updateAll(context.Background())
	assert.NoError(t, err, "no toolchains should be a no-op")
}

func TestUpdateAll_WithInstalledToolchain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	require.NoError(t, config.EnsureDirs())

	// Install first
	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))
	require.NoError(t, InstallToolchainWithOptions(context.Background(), "lts", false))

	// Update all — should check for updates (already latest)
	_, _, err := updateAll(context.Background())
	assert.NoError(t, err)
}

func TestReinstallChannel_AlreadyUpToDate(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	settingsPath := filepath.Join(home, "settings.toml")
	require.NoError(t, config.SaveSettings(&settings, settingsPath))
	require.NoError(t, InstallToolchainWithOptions(context.Background(), "lts", false))

	// Reload settings after install
	settings2, _ := config.LoadSettings(settingsPath)

	// Reinstall same channel — should print "already up to date"
	sf := config.NewSettingsFile(settingsPath)
	err := reinstallChannel(context.Background(), toolchain.LTS, "lts-1.0.5", settings2, sf, nil)
	assert.NoError(t, err)
}

func TestUpdateSingle_ChannelName(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))
	require.NoError(t, InstallToolchainWithOptions(context.Background(), "lts", false))

	// Update by channel name
	err := updateSingle(context.Background(), "lts")
	assert.NoError(t, err)
}

func TestUpdateSingle_TargetVariantUpdatesVariant(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	require.NoError(t, config.EnsureDirs())

	targetKey, err := dist.CurrentPlatformKeyWithTarget("", "ohos")
	require.NoError(t, err)
	oldName := "sts-1.0.0-" + targetKey
	oldDir := filepath.Join(home, "toolchains", oldName)
	require.NoError(t, os.MkdirAll(oldDir, 0o755))

	server := mockServerWithTargetSDKs(t, toolchain.STS, "2.0.0", "ohos")
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	require.NoError(t, updateSingle(context.Background(), oldName))

	installed, err := toolchain.ListInstalled()
	require.NoError(t, err)
	assert.Contains(t, installed, "sts-2.0.0-"+targetKey)
	assert.NotContains(t, installed, oldName)
}

func TestFindInstalledForChannel_Nightly(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	tcDir := filepath.Join(home, "toolchains")
	require.NoError(t, os.MkdirAll(filepath.Join(tcDir, "nightly-20260301"), 0o755))

	name, err := findInstalledForChannel(toolchain.Nightly)
	require.NoError(t, err)
	assert.Equal(t, "nightly-20260301", name)
}

func TestRunUpdate_WithSpecificName(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))
	require.NoError(t, InstallToolchainWithOptions(context.Background(), "lts", false))

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := runUpdate(cmd, []string{"lts"})
	assert.NoError(t, err)
}

func TestReinstallChannel_UpgradesToNewerVersion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	require.NoError(t, config.EnsureDirs())

	// Fake an old installed version (just a directory)
	oldDir := filepath.Join(home, "toolchains", "lts-1.0.0")
	require.NoError(t, os.MkdirAll(oldDir, 0o755))

	// Mock server says latest is 1.0.5
	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	settings.DefaultToolchain = "lts-1.0.0"
	settingsPath := filepath.Join(home, "settings.toml")
	require.NoError(t, config.SaveSettings(&settings, settingsPath))

	// Reinstall should upgrade from 1.0.0 to 1.0.5
	sf := config.NewSettingsFile(settingsPath)
	err := reinstallChannel(context.Background(), toolchain.LTS, "lts-1.0.0", &settings, sf, nil)
	require.NoError(t, err)

	// New version should be installed
	installed, _ := toolchain.ListInstalled()
	assert.Contains(t, installed, "lts-1.0.5")

	// Old version should be removed
	_, statErr := os.Stat(oldDir)
	assert.True(t, os.IsNotExist(statErr), "old version directory should be removed")
}

func TestReinstallChannelForPlatform_UpdatesTargetVariant(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	require.NoError(t, config.EnsureDirs())

	targetKey, err := dist.CurrentPlatformKeyWithTarget("", "ohos")
	require.NoError(t, err)
	oldName := "sts-1.0.0-" + targetKey
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", oldName), 0o755))

	server := mockServerWithTargetSDKs(t, toolchain.STS, "2.0.0", "ohos")
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	settingsPath := filepath.Join(home, "settings.toml")
	require.NoError(t, config.SaveSettings(&settings, settingsPath))

	sf := config.NewSettingsFile(settingsPath)
	require.NoError(t, reinstallChannelForPlatform(context.Background(), toolchain.STS, oldName, &settings, sf, nil, targetKey))

	installed, err := toolchain.ListInstalled()
	require.NoError(t, err)
	assert.Contains(t, installed, "sts-2.0.0-"+targetKey)
	assert.NotContains(t, installed, "sts-2.0.0")
	assert.NotContains(t, installed, oldName)
}

func TestReinstallChannel_UpdatesDefaultToolchain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	require.NoError(t, config.EnsureDirs())

	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.0"), 0o755))

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	settings.DefaultToolchain = "lts-1.0.0"
	settingsPath := filepath.Join(home, "settings.toml")
	require.NoError(t, config.SaveSettings(&settings, settingsPath))

	sf2 := config.NewSettingsFile(settingsPath)
	require.NoError(t, reinstallChannel(context.Background(), toolchain.LTS, "lts-1.0.0", &settings, sf2, nil))

	// Default should be updated to new version
	reloaded, _ := config.LoadSettings(filepath.Join(home, "settings.toml"))
	assert.Equal(t, "lts-1.0.5", reloaded.DefaultToolchain,
		"default should be updated from old version to new version")
}

func TestReinstallChannel_UpdatesOverrides(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	require.NoError(t, config.EnsureDirs())

	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.0"), 0o755))

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	settings.Overrides["C:\\project-a"] = "lts-1.0.0"
	settings.Overrides["C:\\project-b"] = "sts-2.0.0" // different channel, keep
	settingsPath := filepath.Join(home, "settings.toml")
	require.NoError(t, config.SaveSettings(&settings, settingsPath))

	sf3 := config.NewSettingsFile(settingsPath)
	require.NoError(t, reinstallChannel(context.Background(), toolchain.LTS, "lts-1.0.0", &settings, sf3, nil))

	reloaded, _ := config.LoadSettings(filepath.Join(home, "settings.toml"))
	assert.Equal(t, "lts-1.0.5", reloaded.Overrides["C:\\project-a"],
		"override should be updated to new version")
	assert.Equal(t, "sts-2.0.0", reloaded.Overrides["C:\\project-b"],
		"unrelated override should be preserved")
}

// Tests for runUpdate -- cobra handler for update command.

func TestRunUpdate_NoArgs(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	cmd := &cobra.Command{}
	err := runUpdate(cmd, nil)
	assert.NoError(t, err, "update with no toolchains should be a no-op")
}

func TestRunUpdate_WithToolchain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))
	require.NoError(t, InstallToolchainWithOptions(context.Background(), "lts", false))

	cmd := &cobra.Command{}
	err := runUpdate(cmd, nil)
	assert.NoError(t, err)
}

func TestRunUpdate_SingleToolchain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))
	require.NoError(t, InstallToolchainWithOptions(context.Background(), "lts", false))

	cmd := &cobra.Command{}
	err := runUpdate(cmd, []string{"lts-1.0.5"})
	assert.NoError(t, err)
}

// Test updateSingle — updates a single named toolchain.

func TestUpdateSingle_ExistingToolchain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	require.NoError(t, config.EnsureDirs())

	// Install a toolchain first
	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))
	require.NoError(t, InstallToolchainWithOptions(context.Background(), "lts", false))

	// Update the installed toolchain
	err := updateSingle(context.Background(), "lts-1.0.5")
	assert.NoError(t, err)
}

func TestUpdateSingle_UnknownToolchain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	err := updateSingle(context.Background(), "nonexistent-99.99")
	assert.Error(t, err)
}

// Tests for findInstalledForChannel -- used by the update command to
// find which version of a channel is currently installed.

func TestFindInstalledForChannel_FindsLatest(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	tcDir := filepath.Join(home, "toolchains")
	require.NoError(t, os.MkdirAll(filepath.Join(tcDir, "lts-1.0.0"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tcDir, "lts-1.0.5"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tcDir, "sts-2.0.0"), 0o755))

	name, err := findInstalledForChannel(toolchain.LTS)
	require.NoError(t, err)
	assert.Equal(t, "lts-1.0.5", name,
		"should return the latest semver version for the channel")
}

func TestFindInstalledForChannel_IgnoresTargetVariants(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	targetKey, err := dist.CurrentPlatformKeyWithTarget("", "ohos")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.5-"+targetKey), 0o755))

	name, err := findInstalledForChannel(toolchain.LTS)
	require.NoError(t, err)
	assert.Empty(t, name)
}

func TestFindInstalledForChannel_ChannelNotInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.5"), 0o755))

	name, err := findInstalledForChannel(toolchain.STS)
	require.NoError(t, err)
	assert.Empty(t, name, "should return empty string when channel has no installs")
}

func TestFindInstalledForChannel_NoToolchainsDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	name, err := findInstalledForChannel(toolchain.LTS)
	require.NoError(t, err)
	assert.Empty(t, name)
}
