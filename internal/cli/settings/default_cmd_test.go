package settings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/cli/clitest"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for runDefault — sets, shows, or clears the default toolchain.

func TestRunDefault_SetsDefault(t *testing.T) {
	tmp := t.TempDir()
	configtest.Isolate(t, tmp)

	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "toolchains", "lts-1.0.5"), 0o755))
	settings := config.DefaultSettings()
	settingsPath, err := config.SettingsPath()
	require.NoError(t, err)
	require.NoError(t, config.SaveSettings(&settings, settingsPath))

	err = executeSettings(t, "default", "lts-1.0.5")
	require.NoError(t, err)

	reloaded, _ := config.LoadSettings(settingsPath)
	assert.Equal(t, "lts-1.0.5", reloaded.DefaultToolchain)
}

func TestRunDefault_RejectsTargetVariant(t *testing.T) {
	tmp := t.TempDir()
	configtest.Isolate(t, tmp)

	name := "sts-2.0.0-win32-x64-ohos"
	require.NoError(t, os.MkdirAll(filepath.Join(tmp, "toolchains", name), 0o755))
	settings := config.DefaultSettings()
	settingsPath, err := config.SettingsPath()
	require.NoError(t, err)
	require.NoError(t, config.SaveSettings(&settings, settingsPath))

	err = executeSettings(t, "default", name)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "target variant")

	reloaded, err := config.LoadSettings(settingsPath)
	require.NoError(t, err)
	assert.Empty(t, reloaded.DefaultToolchain)
}

func TestRunDefault_ClearsDefault(t *testing.T) {
	tmp := t.TempDir()
	configtest.Isolate(t, tmp)

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	settingsPath, err := config.SettingsPath()
	require.NoError(t, err)
	require.NoError(t, config.SaveSettings(&settings, settingsPath))

	err = executeSettings(t, "default", "none")
	require.NoError(t, err)

	reloaded, _ := config.LoadSettings(settingsPath)
	assert.Empty(t, reloaded.DefaultToolchain)
}

func TestRunDefault_ShowsCurrentDefault(t *testing.T) {
	tmp := t.TempDir()
	configtest.Isolate(t, tmp)

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	settingsPath, err := config.SettingsPath()
	require.NoError(t, err)
	require.NoError(t, config.SaveSettings(&settings, settingsPath))

	err = executeSettings(t, "default")
	assert.NoError(t, err)
}

func TestShowDefault_WithDefaultSet(t *testing.T) {
	tmp := t.TempDir()
	configtest.Isolate(t, tmp)

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	settingsPath, err := config.SettingsPath()
	require.NoError(t, err)
	require.NoError(t, config.SaveSettings(&settings, settingsPath))

	// Should succeed without error
	err = executeSettings(t, "default")
	assert.NoError(t, err)
}

func TestShowDefault_NoDefaultSet(t *testing.T) {
	tmp := t.TempDir()
	configtest.Isolate(t, tmp)

	settings := config.DefaultSettings()
	settings.DefaultToolchain = ""
	settingsPath, err := config.SettingsPath()
	require.NoError(t, err)
	require.NoError(t, config.SaveSettings(&settings, settingsPath))

	// Should succeed (prints "no default" message, not an error)
	err = executeSettings(t, "default")
	assert.NoError(t, err)
}

func TestShowDefault_NoSettingsFile(t *testing.T) {
	tmp := t.TempDir()
	configtest.Isolate(t, tmp)

	// No settings file — LoadSettings returns defaults
	err := executeSettings(t, "default")
	assert.NoError(t, err)
}

func TestDefaultInstallsMissingSDKAndPreservesLegacyOffline(t *testing.T) {
	server := testutil.ValidMockServer(t)
	home := clitest.UseManifest(t, server.URL)
	run := func(args ...string) (string, error) {
		return clitest.Execute(t, clitest.Root(func(root *cobra.Command, _ *clictx.Context) { RegisterCommands(root) }), args...)
	}
	text, err := run("default", "lts", "--json", "--quiet")
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(text)))
	require.DirExists(t, filepath.Join(home, "toolchains", "lts"))
	server.Close()
	legacy := filepath.Join(home, "toolchains", "sts-1.0.0")
	require.NoError(t, os.MkdirAll(legacy, 0o755))
	// Simulate a published layout before the one-time migration marker.
	require.NoError(t, os.Remove(filepath.Join(home, ".cjv", "layout-v2")))
	text, err = run("default", "sts", "--json")
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(text)))
}
