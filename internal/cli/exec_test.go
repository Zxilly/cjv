package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecRun_NoArgs(t *testing.T) {
	cmd := &cobra.Command{}
	err := execRun(cmd, nil)
	assert.Error(t, err, "should error with no arguments")
}

func TestExecRun_NoToolchain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	t.Setenv("CJV_TOOLCHAIN", "")
	config.ResetDefaultSettingsFileCache()
	config.ResetCachedUserHomeDir()

	settings := config.DefaultSettings()
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	t.Chdir(t.TempDir())

	cmd := &cobra.Command{}
	err := execRun(cmd, []string{"./some_binary"})
	assert.Error(t, err, "should error when no toolchain configured")
}

func TestExtractPlusToolchainFromExecArgs(t *testing.T) {
	tests := []struct {
		args       []string
		wantTC     string
		wantRemain []string
	}{
		{[]string{"+nightly", "./binary", "arg"}, "nightly", []string{"./binary", "arg"}},
		{[]string{"./binary", "arg"}, "", []string{"./binary", "arg"}},
		{[]string{"+lts-1.0.5", "cmd"}, "lts-1.0.5", []string{"cmd"}},
	}

	for _, tt := range tests {
		tc, remain := extractPlusToolchainFromArgs(tt.args)
		assert.Equal(t, tt.wantTC, tc)
		assert.Equal(t, tt.wantRemain, remain)
	}
}

func TestExecRunExecutesCommandWithActiveToolchain(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)
	t.Setenv(config.EnvToolchain, "")
	config.ResetDefaultSettingsFileCache()
	require.NoError(t, config.EnsureDirs())
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.5"), 0o755))

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	require.NoError(t, execRun(cmd, []string{"go", "version"}))
	require.NoError(t, execRun(cmd, []string{"--help"}))
}
