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

func TestLookPathInEnv_ExtensionHandling(t *testing.T) {
	dir := t.TempDir()
	environ := []string{"PATH=" + dir}

	if runtime.GOOS == "windows" {
		// A same-named extensionless file must NOT shadow the real .exe.
		require.NoError(t, os.WriteFile(filepath.Join(dir, "tool"), []byte("data"), 0o644))
		exe := filepath.Join(dir, "tool.exe")
		require.NoError(t, os.WriteFile(exe, []byte("MZ"), 0o644))
		environ = append(environ, "PATHEXT=.COM;.EXE;.BAT;.CMD")

		got, found := lookPathInEnv("tool", environ)
		require.True(t, found)
		assert.Equal(t, exe, got)
	} else {
		exe := filepath.Join(dir, "tool")
		require.NoError(t, os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755))

		got, found := lookPathInEnv("tool", environ)
		require.True(t, found)
		assert.Equal(t, exe, got)
	}

	// A command not present on the env PATH is reported as not found.
	got, found := lookPathInEnv("definitely-not-present-xyz", environ)
	assert.False(t, found)
	assert.Equal(t, "definitely-not-present-xyz", got)
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
