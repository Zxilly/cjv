package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for runUpdate, the cobra handler for the update command: argument
// handling, the self-update decision and result rendering. Update semantics
// live with lifecycle.UpdateInstalled / UpdateAll.

func TestRunUpdate_NoArgs(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)

	cmd := &cobra.Command{}
	err := app.runUpdate(cmd, nil)
	assert.NoError(t, err, "update with no toolchains should be a no-op")
}

func TestRunUpdate_WithToolchain(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))
	require.NoError(t, app.InstallToolchainWithOptions(context.Background(), "lts", false))

	cmd := &cobra.Command{}
	err := app.runUpdate(cmd, nil)
	assert.NoError(t, err)
}

func TestRunUpdateRunsSelfCheckWhenSettingsAvailable(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "local-sdk"), 0o755))
	settings := config.DefaultSettings()
	settings.AutoSelfUpdate = config.AutoSelfUpdateCheck
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	app.noSelfUpdate = false

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	require.NoError(t, app.runUpdate(cmd, nil))
}

func TestRunUpdate_WithSpecificName(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))
	require.NoError(t, app.InstallToolchainWithOptions(context.Background(), "lts", false))

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := app.runUpdate(cmd, []string{"lts"})
	assert.NoError(t, err)
}

func TestRunUpdate_SingleToolchain(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))
	require.NoError(t, app.InstallToolchainWithOptions(context.Background(), "lts", false))

	cmd := &cobra.Command{}
	err := app.runUpdate(cmd, []string{"lts-1.0.5"})
	assert.NoError(t, err)
}

func TestRunUpdateRejectsInvalidAndUnknownNames(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)

	cmd := &cobra.Command{}
	require.Error(t, app.runUpdate(cmd, []string{"+bad"}))
	require.Error(t, app.runUpdate(cmd, []string{"local-sdk"}))
	require.Error(t, app.runUpdate(cmd, []string{"nonexistent-99.99"}))
}

func TestRunUpdateRendersJSONResult(t *testing.T) {
	app := newApplication("dev", "")
	app.output.SetJSONMode(true)
	app.noSelfUpdate = true
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.0"), 0o755))
	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&out)
	require.NoError(t, app.runUpdate(cmd, nil))

	var result struct {
		Updates       []updateEntry `json:"updates"`
		NoneInstalled bool          `json:"none_installed"`
	}
	require.NoError(t, json.Unmarshal(out.Bytes(), &result), out.String())
	assert.Equal(t, []updateEntry{{From: "lts-1.0.0", To: "lts-1.0.5"}}, result.Updates)
	assert.False(t, result.NoneInstalled)
}

func TestUpdateEntriesKeepsAppliedOutcomesOnly(t *testing.T) {
	entries := updateEntries([]lifecycle.UpdateOutcome{
		{Name: "local-sdk", Status: lifecycle.UpdateSkipped},
		{Name: "lts-1.0.5", Replacement: "lts-1.0.5", Status: lifecycle.UpdateUpToDate},
		{Name: "sts-1.0.0", Replacement: "sts-2.0.0", Status: lifecycle.UpdateApplied},
		{Name: "nightly-1", Status: lifecycle.UpdateFailed},
	})
	assert.Equal(t, []updateEntry{{From: "sts-1.0.0", To: "sts-2.0.0"}}, entries)
	assert.Nil(t, updateEntries(nil))
}
