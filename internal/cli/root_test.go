package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test for Execute — the root command entry point.

func TestExecute_Help(t *testing.T) {
	oldArgs := os.Args
	os.Args = []string{"cjv", "--help"}
	defer func() { os.Args = oldArgs }()

	err := Execute("dev", "dev")
	assert.NoError(t, err)
}

func TestRootCommandRunListsInstalledAndMarksActive(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.5"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "sts-2.0.0"), 0o755))
	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	stdout, err := captureStdout(t, func() error {
		return rootCmd.RunE(rootCmd, nil)
	})

	require.NoError(t, err)
	assert.Contains(t, stdout, "lts-1.0.5")
	assert.Contains(t, stdout, "sts-2.0.0")
	assert.True(t, strings.Contains(stdout, "* lts-1.0.5") || strings.Contains(stdout, "*  lts-1.0.5"))
}

func TestRootCommandRunWithNoToolchains(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)

	stdout, err := captureStdout(t, func() error {
		return rootCmd.RunE(rootCmd, nil)
	})

	require.NoError(t, err)
	assert.Contains(t, stdout, "cjv install lts")
}
