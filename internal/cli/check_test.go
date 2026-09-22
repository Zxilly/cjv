package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for runCheck — checks all installed toolchains for updates
// by comparing installed versions against the manifest.

func TestRunCheck_NoToolchains(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)

	cmd := &cobra.Command{}
	err := app.runCheck(cmd, nil)
	assert.NoError(t, err, "no toolchains should be a no-op, not error")
}

func TestRunCheck_WithInstalledToolchain(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	// Install a toolchain first
	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))
	require.NoError(t, app.InstallToolchainWithOptions(context.Background(), "lts", false))

	// Run check — should compare against manifest
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := app.runCheck(cmd, nil)
	assert.NoError(t, err)
}

func TestRunCheck_NightlyUsesUnifiedDistServer(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	server := splitNightlyMockServer(t)
	settings := config.DefaultSettings()
	settings.DistServer = server.URL + "/corp/cjv"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))
	require.NoError(t, app.InstallToolchainWithOptions(context.Background(), "nightly", false))

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	app.output.SetJSONMode(true)

	require.NoError(t, app.runCheck(cmd, nil))

	var got checkResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got.Toolchains, 1)
	assert.Empty(t, got.Toolchains[0].Error)
	assert.Equal(t, "nightly-1.2.0-alpha.20260822010101", got.Toolchains[0].Latest)
}

func TestRunCheck_UpToDate(t *testing.T) {
	// Install latest, then check — should show "all up to date"
	app := newApplication("dev", "")

	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	require.NoError(t, app.InstallToolchainWithOptions(context.Background(), "lts", false))
	require.NoError(t, app.InstallToolchainWithOptions(context.Background(), "sts", false))

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := app.runCheck(cmd, nil)
	assert.NoError(t, err)
}

func TestRunCheck_MixedVersions(t *testing.T) {
	// One channel up to date, one outdated
	app := newApplication("dev", "")

	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	// Install latest lts
	require.NoError(t, app.InstallToolchainWithOptions(context.Background(), "lts", false))
	// Create an old sts
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "sts-1.0.0"), 0o755))

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := app.runCheck(cmd, nil)
	assert.NoError(t, err)
}

func TestRunCheck_UpdateAvailable(t *testing.T) {
	// Install an "old" version by creating the directory manually,
	// then check against the manifest that has a newer version.
	app := newApplication("dev", "")

	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	// Create a fake old installed LTS toolchain
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.0"), 0o755))

	// Mock server says latest LTS is 1.0.5
	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := app.runCheck(cmd, nil)
	assert.NoError(t, err, "check should succeed even when updates are available")
}

func TestRunCheck_MultipleToolchains(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	// Create both LTS and STS old versions
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.0"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "sts-1.0.0"), 0o755))

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := app.runCheck(cmd, nil)
	assert.NoError(t, err)
}

func TestRunCheck_CustomToolchainSkipped(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	// Custom toolchains (non-standard names) should be skipped by check
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "my-custom-sdk"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.0"), 0o755))

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := app.runCheck(cmd, nil)
	assert.NoError(t, err)
}
