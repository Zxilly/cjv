package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFakeToolchainForCLI(t *testing.T, home, name string) {
	t.Helper()
	tcDir := filepath.Join(home, "toolchains", name)
	binDir := filepath.Join(tcDir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	cjcName := "cjc"
	if runtime.GOOS == "windows" {
		cjcName = "cjc.exe"
	}
	require.NoError(t, os.WriteFile(filepath.Join(binDir, cjcName), []byte("stub"), 0o755))
}

func TestEnvsetupRun_NoToolchain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	t.Setenv("CJV_TOOLCHAIN", "")
	config.ResetDefaultSettingsFileCache()
	config.ResetCachedUserHomeDir()

	settings := config.DefaultSettings()
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	t.Chdir(t.TempDir())

	cmd := &cobra.Command{}
	err := envsetupRun(cmd, nil)
	assert.Error(t, err)
}

func TestEnvsetupRun_OutputContainsExport(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	t.Setenv("CJV_TOOLCHAIN", "")
	config.ResetDefaultSettingsFileCache()
	config.ResetCachedUserHomeDir()

	setupFakeToolchainForCLI(t, home, "lts-1.0.5")
	// Also create bin dir for cjv
	require.NoError(t, os.MkdirAll(filepath.Join(home, "bin"), 0o755))

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	t.Chdir(t.TempDir())

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetOut(&buf)

	err := envsetupRunWithShell(cmd, nil, "bash")
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "export ")
	assert.Contains(t, output, "lts-1.0.5")
}
