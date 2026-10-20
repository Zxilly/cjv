package settingscmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/cli/clitest"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for the default command: it sets, shows, or clears the default toolchain.

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

func TestShowDefault(t *testing.T) {
	for _, tc := range []struct {
		name         string
		settings     bool
		defaultName  string
		wantText     string
		wantJSONName string
	}{
		{name: "set", settings: true, defaultName: "lts-1.0.5", wantText: "lts-1.0.5", wantJSONName: "lts-1.0.5"},
		{name: "unset", settings: true, wantText: i18n.T("NoDefaultToolchain", nil)},
		{name: "no settings file", wantText: i18n.T("NoDefaultToolchain", nil)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			configtest.Isolate(t, t.TempDir())
			if tc.settings {
				settings := config.DefaultSettings()
				settings.DefaultToolchain = tc.defaultName
				settingsPath, err := config.SettingsPath()
				require.NoError(t, err)
				require.NoError(t, config.SaveSettings(&settings, settingsPath))
			}

			out, err := executeSettingsOutput(t, "default")
			require.NoError(t, err, "showing the default is never an error")
			assert.Contains(t, out, tc.wantText)

			out, err = executeSettingsOutput(t, "default", "--json")
			require.NoError(t, err)
			var result struct {
				Toolchain string `json:"default_toolchain"`
			}
			require.NoError(t, json.Unmarshal([]byte(out), &result), out)
			assert.Equal(t, tc.wantJSONName, result.Toolchain)
		})
	}
}

func TestDefaultInstallsMissingSDKAndPreservesLegacyOffline(t *testing.T) {
	server := testutil.ValidMockServer(t)
	home := clitest.UseManifest(t, server.URL)
	run := func(args ...string) (string, error) {
		return clitest.Execute(t, clitest.Root(Register), args...)
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
