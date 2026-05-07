package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunInitNonInteractiveNoToolchainWritesManagedFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)
	config.ResetDefaultSettingsFileCache()

	oldYes := initYes
	oldToolchain := initDefaultToolchain
	oldNoModifyPath := initNoModifyPath
	initYes = true
	initDefaultToolchain = "none"
	initNoModifyPath = true
	t.Cleanup(func() {
		initYes = oldYes
		initDefaultToolchain = oldToolchain
		initNoModifyPath = oldNoModifyPath
		config.ResetDefaultSettingsFileCache()
	})

	err := runInit(&cobra.Command{}, nil)

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(home, "bin", proxy.CjvBinaryName()))
	if runtime.GOOS == "windows" {
		assert.FileExists(t, filepath.Join(home, "env.ps1"))
		assert.FileExists(t, filepath.Join(home, "env.bat"))
	} else {
		assert.FileExists(t, filepath.Join(home, "env"))
	}
	for _, tool := range proxy.AllProxyTools() {
		assert.FileExists(t, filepath.Join(home, "bin", proxy.PlatformBinaryName(tool)))
	}

	settings, err := config.LoadSettings(filepath.Join(home, "settings.toml"))
	require.NoError(t, err)
	assert.Equal(t, config.DefaultManifestURL, settings.ManifestURL)
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "lts-1.0.5"))
}

func TestRunInitContinuesWhenDefaultToolchainInstallFails(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)
	config.ResetDefaultSettingsFileCache()

	oldYes := initYes
	oldToolchain := initDefaultToolchain
	oldNoModifyPath := initNoModifyPath
	initYes = true
	initDefaultToolchain = "local-sdk"
	initNoModifyPath = true
	t.Cleanup(func() {
		initYes = oldYes
		initDefaultToolchain = oldToolchain
		initNoModifyPath = oldNoModifyPath
		_ = os.Unsetenv(config.EnvNoPathSetup)
		config.ResetDefaultSettingsFileCache()
	})

	err := runInit(&cobra.Command{}, nil)

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(home, "bin", proxy.CjvBinaryName()))
	assert.Empty(t, os.Getenv(config.EnvNoPathSetup))
}

func TestRunInitCoversAlreadyInstalledAndModifyPathBranches(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)
	config.ResetDefaultSettingsFileCache()

	oldYes := initYes
	oldToolchain := initDefaultToolchain
	oldNoModifyPath := initNoModifyPath
	oldEnsurePath := ensurePathConfiguredFn
	var pathConfigured bool
	initYes = true
	initDefaultToolchain = "none"
	initNoModifyPath = false
	ensurePathConfiguredFn = func() { pathConfigured = true }
	t.Cleanup(func() {
		initYes = oldYes
		initDefaultToolchain = oldToolchain
		initNoModifyPath = oldNoModifyPath
		ensurePathConfiguredFn = oldEnsurePath
		config.ResetDefaultSettingsFileCache()
	})

	require.NoError(t, runInit(&cobra.Command{}, nil))
	require.True(t, pathConfigured)

	pathConfigured = false
	require.NoError(t, runInit(&cobra.Command{}, nil))
	require.True(t, pathConfigured)

	assert.NotEmpty(t, yesNoStr(true))
	assert.NotEmpty(t, yesNoStr(false))
}
