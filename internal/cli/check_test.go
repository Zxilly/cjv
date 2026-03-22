package cli

import (
	"context"
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
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	cmd := &cobra.Command{}
	err := runCheck(cmd, nil)
	assert.NoError(t, err, "no toolchains should be a no-op, not error")
}

func TestRunCheck_WithInstalledToolchain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	require.NoError(t, config.EnsureDirs())

	// Install a toolchain first
	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))
	require.NoError(t, InstallToolchainWithOptions(context.Background(), "lts", false))

	// Run check — should compare against manifest
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := runCheck(cmd, nil)
	assert.NoError(t, err)
}

func TestRunCheck_UpToDate(t *testing.T) {
	// Install latest, then check — should show "all up to date"
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	require.NoError(t, InstallToolchainWithOptions(context.Background(), "lts", false))
	require.NoError(t, InstallToolchainWithOptions(context.Background(), "sts", false))

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := runCheck(cmd, nil)
	assert.NoError(t, err)
}

func TestRunCheck_MixedVersions(t *testing.T) {
	// One channel up to date, one outdated
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	// Install latest lts
	require.NoError(t, InstallToolchainWithOptions(context.Background(), "lts", false))
	// Create an old sts
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "sts-1.0.0"), 0o755))

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := runCheck(cmd, nil)
	assert.NoError(t, err)
}

func TestRunCheck_UpdateAvailable(t *testing.T) {
	// Install an "old" version by creating the directory manually,
	// then check against the manifest that has a newer version.
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	require.NoError(t, config.EnsureDirs())

	// Create a fake old installed LTS toolchain
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.0"), 0o755))

	// Mock server says latest LTS is 1.0.5
	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := runCheck(cmd, nil)
	assert.NoError(t, err, "check should succeed even when updates are available")
}

func TestRunCheck_MultipleToolchains(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	require.NoError(t, config.EnsureDirs())

	// Create both LTS and STS old versions
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.0"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "sts-1.0.0"), 0o755))

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := runCheck(cmd, nil)
	assert.NoError(t, err)
}

func TestRunCheck_CustomToolchainSkipped(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	require.NoError(t, config.EnsureDirs())

	// Custom toolchains (non-standard names) should be skipped by check
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "my-custom-sdk"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.0"), 0o755))

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := runCheck(cmd, nil)
	assert.NoError(t, err)
}
