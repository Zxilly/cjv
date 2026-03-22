package cli

import (
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
	t.Setenv("CJV_HOME", home)
	t.Setenv("CJV_TOOLCHAIN", "")

	t.Chdir(cwd)

	settings := config.DefaultSettings()
	settings.AutoInstall = false
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	cmd := &cobra.Command{}
	err := runRun(cmd, []string{"cjc", "--version"})
	assert.Error(t, err, "should error when no toolchain is configured")
}
