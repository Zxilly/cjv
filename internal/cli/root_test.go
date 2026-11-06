package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRootAndShowShareStatus(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	t.Chdir(t.TempDir())
	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))
	require.NoError(t, toolchain.WriteInstallation(filepath.Join(home, "toolchains", "lts"), toolchain.Installation{Release: "lts-1.0.5"}))
	for _, format := range [][]string{nil, {"--json"}} {
		bare, err := executeWithOutput(t, newApplication("dev", ""), format)
		require.NoError(t, err)
		show, err := executeWithOutput(t, newApplication("dev", ""), append([]string{"show"}, format...))
		require.NoError(t, err)
		assert.Equal(t, show, bare)
		assert.Contains(t, bare, "lts-1.0.5", "the installed tracking release is visible")
	}
}

func TestCommandsRejectUnexpectedArguments(t *testing.T) {
	configtest.Isolate(t, t.TempDir())
	t.Chdir(t.TempDir())
	for _, args := range [][]string{
		{"show", "active", "typo"}, {"show", "installed", "typo"}, {"show", "home", "typo"},
		{"toolchain", "list", "nightly"}, {"override", "unset", "typo"}, {"override", "list", "typo"},
		{"check", "typo"}, {"self", "update", "typo"}, {"self", "uninstall", "typo"}, {"init", "typo"},
	} {
		t.Run(strings.Join(args, "/"), func(t *testing.T) {
			_, err := executeWithOutput(t, newApplication("dev", ""), args)
			require.Error(t, err)
			assert.Contains(t, err.Error(), args[len(args)-1], "all rejection happens before any operation")
		})
	}
}

func TestCanonicalHelpKeepsCompatibilityEntrypoints(t *testing.T) {
	configtest.Isolate(t, t.TempDir())
	app := newApplication("dev", "")
	for _, path := range [][]string{{"show", "installed"}, {"toolchain", "uninstall"}} {
		cmd, rest, err := app.rootCmd.Find(path)
		require.NoError(t, err)
		require.Empty(t, rest)
		assert.True(t, cmd.Hidden)
	}
	assert.Len(t, app.rootCmd.Groups(), 5)
}

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
	configtest.Isolate(t, t.TempDir())
	for _, args := range [][]string{{"--json"}, {}} {
		app := newApplication("dev", "")
		var stdout, stderr bytes.Buffer
		app.rootCmd.SetOut(&stdout)
		app.rootCmd.SetErr(&stderr)
		require.NoError(t, app.execute(args))
		if len(args) > 0 {
			assert.Empty(t, stderr.String())
			assert.True(t, json.Valid(stdout.Bytes()), stdout.String())
		} else {
			assert.False(t, json.Valid(stdout.Bytes()), stdout.String())
			assert.Contains(t, stdout.String(), "cjv install lts")
		}
	}
}

func TestExecuteArgumentFailureDoesNotAffectNextInvocation(t *testing.T) {
	configtest.Isolate(t, t.TempDir())
	failed := newApplication("dev", "")
	var failureOutput, failureErrors bytes.Buffer
	failed.rootCmd.SetOut(&failureOutput)
	failed.rootCmd.SetErr(&failureErrors)
	require.Error(t, failed.execute([]string{"--json", "install", "+invalid"}))
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

func TestRootCommandRunListsInstalledAndActive(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	configtest.Isolate(t, home)
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.5"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "sts-2.0.0"), 0o755))
	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	stdout, err := runWithRootOutput(t, app)

	require.NoError(t, err)
	assert.Contains(t, stdout, "lts-1.0.5")
	assert.Contains(t, stdout, "sts-2.0.0")
}

func TestRootCommandRunWithNoToolchains(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	configtest.Isolate(t, home)

	stdout, err := runWithRootOutput(t, app)

	require.NoError(t, err)
	assert.Contains(t, stdout, "cjv install lts")
}

func TestGlobalSelectorAndBatchJSON(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	for _, name := range []string{"sts-1.0.0", "lts-1.0.5"} {
		require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", name), 0o755))
	}
	sf, err := config.DefaultSettingsFile()
	require.NoError(t, err)
	name := "lts-1.0.5"
	_, err = sf.Update(config.SettingsUpdate{DefaultToolchain: &name})
	require.NoError(t, err)
	text, err := executeWithOutput(t, newApplication("dev", ""), []string{"+sts-1.0.0", "--json"})
	require.NoError(t, err)
	var status struct {
		Active struct {
			Name string `json:"name"`
		} `json:"active"`
	}
	require.NoError(t, json.Unmarshal([]byte(text), &status))
	require.Equal(t, "sts-1.0.0", status.Active.Name)
	text, err = executeWithOutput(t, newApplication("dev", ""), []string{"uninstall", "sts-1.0", "lts-1.0.5", "--yes", "--json"})
	require.NoError(t, err)
	var removed struct {
		Removed []struct{ Name string } `json:"removed"`
	}
	require.NoError(t, json.Unmarshal([]byte(text), &removed))
	require.Len(t, removed.Removed, 2)
}
