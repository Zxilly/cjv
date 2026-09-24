package cli

import (
	"bytes"
	"encoding/json"
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

func TestExecuteRepeatedInvocations(t *testing.T) {
	oldArgs := os.Args
	t.Cleanup(func() { os.Args = oldArgs })
	for range 2 {
		os.Args = []string{"cjv", "--help"}
		require.NotPanics(t, func() {
			require.NoError(t, Execute("dev", ""))
		})
	}
}

func TestExecuteInvocationsDoNotShareJSONMode(t *testing.T) {
	config.IsolateForTest(t, t.TempDir())
	for _, args := range [][]string{{"--json"}, {}} {
		app := newApplication("dev", "")
		var stdout, stderr bytes.Buffer
		app.rootCmd.SetOut(&stdout)
		app.rootCmd.SetErr(&stderr)
		require.NoError(t, app.execute(args))
		assert.Empty(t, stderr.String())
		if len(args) > 0 {
			assert.True(t, json.Valid(stdout.Bytes()), stdout.String())
		} else {
			assert.False(t, json.Valid(stdout.Bytes()), stdout.String())
			assert.Contains(t, stdout.String(), "cjv install lts")
		}
	}
}

func TestExecuteArgumentFailureDoesNotAffectNextInvocation(t *testing.T) {
	failed := newApplication("dev", "")
	var failureOutput, failureErrors bytes.Buffer
	failed.rootCmd.SetOut(&failureOutput)
	failed.rootCmd.SetErr(&failureErrors)
	require.Error(t, failed.execute([]string{"--json", "install"}))
	assert.True(t, json.Valid(failureOutput.Bytes()), failureOutput.String())
	assert.Empty(t, failureErrors.String())

	next := newApplication("test-version", "")
	var stdout, stderr bytes.Buffer
	next.rootCmd.SetOut(&stdout)
	next.rootCmd.SetErr(&stderr)
	require.NoError(t, next.execute([]string{"--version"}))
	assert.Contains(t, stdout.String(), "cjv test-version")
	assert.False(t, json.Valid(stdout.Bytes()))
	assert.Empty(t, stderr.String())
}

func TestExecuteUnknownCommandJSONDoesNotAffectNextHelp(t *testing.T) {
	failed := newApplication("dev", "")
	var failureOutput, failureErrors bytes.Buffer
	failed.rootCmd.SetOut(&failureOutput)
	failed.rootCmd.SetErr(&failureErrors)
	require.Error(t, failed.execute([]string{"--json", "nonexistent"}))
	var envelope struct {
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal(failureOutput.Bytes(), &envelope))
	assert.Contains(t, envelope.Error.Message, "nonexistent")
	assert.Empty(t, failureErrors.String())

	next := newApplication("dev", "")
	var stdout, stderr bytes.Buffer
	next.rootCmd.SetOut(&stdout)
	next.rootCmd.SetErr(&stderr)
	require.NoError(t, next.execute([]string{"--help"}))
	assert.Contains(t, stdout.String(), "cjv")
	assert.False(t, json.Valid(stdout.Bytes()))
	assert.Empty(t, stderr.String())
}

func TestRootCommandRunListsInstalledAndMarksActive(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.5"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "sts-2.0.0"), 0o755))
	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	stdout, err := runWithRootOutput(t, app)

	require.NoError(t, err)
	assert.Contains(t, stdout, "lts-1.0.5")
	assert.Contains(t, stdout, "sts-2.0.0")
	assert.True(t, strings.Contains(stdout, "* lts-1.0.5") || strings.Contains(stdout, "*  lts-1.0.5"))
}

func TestRootCommandRunWithNoToolchains(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)

	stdout, err := runWithRootOutput(t, app)

	require.NoError(t, err)
	assert.Contains(t, stdout, "cjv install lts")
}
