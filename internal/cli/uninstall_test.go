package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/fstx"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for the uninstall pipeline. These verify that uninstalling a
// toolchain properly removes the directory and updates settings.

func TestRunUninstall_RemovesToolchain(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	// Install first using mock server
	server := testutil.ValidMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))
	installTestToolchain(t, app, "lts")

	// Verify installed
	installed, _ := toolchain.ListInstalled()
	require.NotEmpty(t, installed)

	// Uninstall
	err := app.runUninstall(nil, []string{installed[0]})
	require.NoError(t, err)

	// Verify removed
	remaining, _ := toolchain.ListInstalled()
	assert.Empty(t, remaining)
}

func TestRunUninstall_NotInstalled(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains"), 0o755))

	err := app.runUninstall(nil, []string{"nonexistent-99.99"})
	assert.Error(t, err, "uninstalling non-existent toolchain should error")
}

func TestRunUninstallRecoversBeforeCheckingInstallation(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())
	name := "lts-1.0.5"
	dir := filepath.Join(home, "toolchains", name)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "compiler"), []byte("old compiler"), 0o644))
	tx, err := fstx.NewToolchainTransaction(home, name)
	require.NoError(t, err)
	require.NoError(t, tx.RemoveDir(dir))
	assert.NoDirExists(t, dir)
	// The CLI must recover before its confirmation/existence check, not report
	// "not installed" for content that is still owned by a retained journal.
	require.NoError(t, app.runUninstall(nil, []string{name}))
	assert.NoDirExists(t, dir)
	entries, err := os.ReadDir(filepath.Dir(dir))
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestRunUninstall_PreservesSettingsWhenRemoveFails(t *testing.T) {
	app := newApplication("dev", "")
	if runtime.GOOS != "windows" {
		t.Skip("Windows keeps the process working directory locked")
	}

	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	name := "lts-1.0.5"
	toolchainDir := filepath.Join(home, "toolchains", name)
	require.NoError(t, os.MkdirAll(toolchainDir, 0o755))

	projectDir := filepath.Join(home, "project")
	settings := config.DefaultSettings()
	settings.DefaultToolchain = name
	settings.Overrides[projectDir] = name
	settingsPath := filepath.Join(home, ".cjv", "settings.toml")
	require.NoError(t, config.SaveSettings(&settings, settingsPath))

	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(toolchainDir))
	t.Cleanup(func() {
		_ = os.Chdir(wd)
		_ = os.RemoveAll(filepath.Join(home, "toolchains"))
	})

	err = app.runUninstall(nil, []string{name})

	require.Error(t, err)
	loaded, err := config.LoadSettings(settingsPath)
	require.NoError(t, err)
	assert.Equal(t, name, loaded.DefaultToolchain)
	assert.Equal(t, name, loaded.Overrides[projectDir])
}

func TestUpdateSettingsAfterUninstallDoesNotPromoteTargetVariantToDefault(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	config.ResetDefaultSettingsFileCache()
	t.Cleanup(config.ResetDefaultSettingsFileCache)
	require.NoError(t, config.EnsureDirs())

	name := "lts-1.0.5"
	targetVariant := "lts-1.0.5-linux-x64-ohos"
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", name), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", targetVariant), 0o755))

	settings := config.DefaultSettings()
	settings.DefaultToolchain = name
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	require.NoError(t, lifecycle.RemoveToolchain(name))

	loaded, err := config.LoadSettings(filepath.Join(home, ".cjv", "settings.toml"))
	require.NoError(t, err)
	assert.Empty(t, loaded.DefaultToolchain)
}

func TestRunUninstall_MultipleInstalled(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	server := testutil.ValidMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	// Install lts
	installTestToolchain(t, app, "lts")
	// Create a fake sts toolchain
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "sts-2.0.0"), 0o755))

	installed, _ := toolchain.ListInstalled()
	require.Len(t, installed, 2)

	// Uninstall sts, lts should remain
	err := app.runUninstall(nil, []string{"sts-2.0.0"})
	require.NoError(t, err)

	remaining, _ := toolchain.ListInstalled()
	assert.Len(t, remaining, 1)
	assert.Contains(t, remaining, "lts-1.0.5")
}
