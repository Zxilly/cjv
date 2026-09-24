package cli

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRunCommandPrefersToolchainBinary(t *testing.T) {
	tcDir := t.TempDir()
	binDir := filepath.Join(tcDir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))

	toolPath := filepath.Join(binDir, "cjc")
	if runtime.GOOS == "windows" {
		toolPath += ".exe"
	}
	require.NoError(t, os.WriteFile(toolPath, []byte("stub"), 0o755))

	got, found := resolveToolchainToolPath(tcDir, "cjc")
	assert.True(t, found)
	assert.Equal(t, toolPath, got)
}

func TestResolveRunCommandFallsBackForUnknownCommand(t *testing.T) {
	got, found := resolveToolchainToolPath(t.TempDir(), "powershell")
	assert.False(t, found)
	assert.Equal(t, "powershell", got)
}

func TestResolveRunCommandFallsBackWhenMappedToolIsMissing(t *testing.T) {
	got, found := resolveToolchainToolPath(t.TempDir(), "cjc")
	assert.False(t, found)
	assert.Equal(t, "cjc", got)
}

func TestRunRun_NoToolchain(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	cwd := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv("CJV_TOOLCHAIN", "")

	t.Chdir(cwd)

	settings := config.DefaultSettings()
	settings.AutoInstall = false
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	cmd := &cobra.Command{}
	err := app.runRun(cmd, []string{"cjc", "--version"})
	assert.Error(t, err, "should error when no toolchain is configured")
}

func TestRunRunExecutesFallbackCommandForInstalledToolchain(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv(config.EnvToolchain, "")
	require.NoError(t, config.EnsureDirs())
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.5"), 0o755))
	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	err := app.runRun(cmd, []string{"lts", "go", "version"})

	require.NoError(t, err)
}

func TestRunRunHandlesHelpAndInvalidArgs(t *testing.T) {
	app := newApplication("dev", "")
	cmd := &cobra.Command{Use: "run"}

	require.NoError(t, app.runRun(cmd, []string{"--help"}))
	require.Error(t, app.runRun(cmd, []string{"lts"}))
	require.Error(t, app.runRun(cmd, []string{"bad/name", "go"}))
}
