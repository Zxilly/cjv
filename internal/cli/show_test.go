package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunShowActiveShowsNotInstalledGracefully(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	cwd := t.TempDir()
	config.IsolateForTest(t, home)
	t.Chdir(cwd)

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	// Should succeed (not error) — uninstalled toolchains are shown with annotation
	err := app.runShowActive(app.showActiveCmd, nil)
	require.NoError(t, err)
}

func TestRunShowInstalled_ListsToolchains(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)

	tcDir := filepath.Join(home, "toolchains")
	require.NoError(t, os.MkdirAll(filepath.Join(tcDir, "lts-1.0.5"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tcDir, "sts-2.0.0"), 0o755))

	cmd := &cobra.Command{}
	err := app.runShowInstalled(cmd, nil)
	assert.NoError(t, err)
}

func TestRunShowInstalled_NoToolchains(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)

	cmd := &cobra.Command{}
	err := app.runShowInstalled(cmd, nil)
	assert.NoError(t, err) // prints "no toolchains installed", not error
}

func TestRunShowDefault_WithDefault(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.5"), 0o755))

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	cmd := &cobra.Command{}
	err := app.runShowDefault(cmd, nil)
	assert.NoError(t, err)
}

func TestRunShowDefault_NoToolchains(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)

	var stdout, stderr bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	err := app.runShowDefault(cmd, nil)
	assert.NoError(t, err)
	assert.Contains(t, stderr.String(), (&cjverr.NoToolchainConfiguredError{}).Error(),
		"the missing-default note reaches the command's err writer")
	assert.Contains(t, stdout.String(), i18n.T("NoToolchainsInstalled", nil))

	stdout.Reset()
	stderr.Reset()
	app.output.SetJSONMode(true)
	require.NoError(t, app.runShowDefault(cmd, nil))
	assert.Empty(t, stderr.String(), "JSON mode keeps the note off stderr")
}

func TestRunShowActive_WithActiveToolchain(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	cwd := t.TempDir()
	config.IsolateForTest(t, home)

	t.Chdir(cwd)

	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.5"), 0o755))
	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	cmd := &cobra.Command{}
	err := app.runShowActive(cmd, nil)
	assert.NoError(t, err)
}

func TestRunShowActive_NoActiveToolchain(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	cwd := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv("CJV_TOOLCHAIN", "")

	t.Chdir(cwd)

	settings := config.DefaultSettings()
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	cmd := &cobra.Command{}
	err := app.runShowActive(cmd, nil)
	assert.Error(t, err, "should error when no toolchain is active")
}
