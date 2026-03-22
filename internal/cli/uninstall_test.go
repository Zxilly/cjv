package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for the uninstall pipeline. These verify that uninstalling a
// toolchain properly removes the directory and updates settings.

func TestRunUninstall_RemovesToolchain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	require.NoError(t, config.EnsureDirs())

	// Install first using mock server
	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))
	require.NoError(t, InstallToolchainWithOptions(context.Background(), "lts", false))

	// Verify installed
	installed, _ := toolchain.ListInstalled()
	require.NotEmpty(t, installed)

	// Uninstall
	err := runUninstall(nil, []string{installed[0]})
	require.NoError(t, err)

	// Verify removed
	remaining, _ := toolchain.ListInstalled()
	assert.Empty(t, remaining)
}

func TestRunUninstall_NotInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains"), 0o755))

	err := runUninstall(nil, []string{"nonexistent-99.99"})
	assert.Error(t, err, "uninstalling non-existent toolchain should error")
}

func TestRunUninstall_MultipleInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	// Install lts
	require.NoError(t, InstallToolchainWithOptions(context.Background(), "lts", false))
	// Create a fake sts toolchain
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "sts-2.0.0"), 0o755))

	installed, _ := toolchain.ListInstalled()
	require.Len(t, installed, 2)

	// Uninstall sts, lts should remain
	err := runUninstall(nil, []string{"sts-2.0.0"})
	require.NoError(t, err)

	remaining, _ := toolchain.ListInstalled()
	assert.Len(t, remaining, 1)
	assert.Contains(t, remaining, "lts-1.0.5")
}
