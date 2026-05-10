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

	assert.Equal(t, toolPath, resolveRunCommand(tcDir, "cjc"))
}

func TestResolveRunCommandFallsBackForUnknownCommand(t *testing.T) {
	assert.Equal(t, "powershell", resolveRunCommand(t.TempDir(), "powershell"))
}

func TestResolveRunCommandFallsBackWhenMappedToolIsMissing(t *testing.T) {
	assert.Equal(t, "cjc", resolveRunCommand(t.TempDir(), "cjc"))
}

func TestRunRun_NoToolchain(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv("CJV_TOOLCHAIN", "")

	t.Chdir(cwd)

	settings := config.DefaultSettings()
	settings.AutoInstall = false
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	cmd := &cobra.Command{}
	err := runRun(cmd, []string{"cjc", "--version"})
	assert.Error(t, err, "should error when no toolchain is configured")
}

func TestRunRunExecutesFallbackCommandForInstalledToolchain(t *testing.T) {
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
	err := runRun(cmd, []string{"lts", "go", "version"})

	require.NoError(t, err)
}

func TestRunRunHandlesHelpAndInvalidArgs(t *testing.T) {
	cmd := &cobra.Command{Use: "run"}

	require.NoError(t, runRun(cmd, []string{"--help"}))
	require.Error(t, runRun(cmd, []string{"lts"}))
	require.Error(t, runRun(cmd, []string{"bad/name", "go"}))
}
