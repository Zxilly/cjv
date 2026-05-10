package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunShowActiveShowsNotInstalledGracefully(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	config.IsolateForTest(t, home)
	t.Chdir(cwd)

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	// Should succeed (not error) — uninstalled toolchains are shown with annotation
	err := runShowActive(showActiveCmd, nil)
	require.NoError(t, err)
}

func TestRunShowInstalled_ListsToolchains(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)

	tcDir := filepath.Join(home, "toolchains")
	require.NoError(t, os.MkdirAll(filepath.Join(tcDir, "lts-1.0.5"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tcDir, "sts-2.0.0"), 0o755))

	cmd := &cobra.Command{}
	err := runShowInstalled(cmd, nil)
	assert.NoError(t, err)
}

func TestRunShowInstalled_NoToolchains(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)

	cmd := &cobra.Command{}
	err := runShowInstalled(cmd, nil)
	assert.NoError(t, err) // prints "no toolchains installed", not error
}

func TestRunShowDefault_WithDefault(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.5"), 0o755))

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	cmd := &cobra.Command{}
	err := runShowDefault(cmd, nil)
	assert.NoError(t, err)
}

func TestRunShowDefault_NoToolchains(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)

	cmd := &cobra.Command{}
	err := runShowDefault(cmd, nil)
	assert.NoError(t, err)
}

func TestRunShowActive_WithActiveToolchain(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	config.IsolateForTest(t, home)

	t.Chdir(cwd)

	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.5"), 0o755))
	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	cmd := &cobra.Command{}
	err := runShowActive(cmd, nil)
	assert.NoError(t, err)
}

func TestRunShowActive_NoActiveToolchain(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv("CJV_TOOLCHAIN", "")

	t.Chdir(cwd)

	settings := config.DefaultSettings()
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	cmd := &cobra.Command{}
	err := runShowActive(cmd, nil)
	assert.Error(t, err, "should error when no toolchain is active")
}
