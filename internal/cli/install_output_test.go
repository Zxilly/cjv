package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/stretchr/testify/require"
)

func setupComponentOutputTest(t *testing.T) (string, string) {
	t.Helper()
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv(config.EnvNoPathSetup, "1")
	t.Setenv(config.EnvDistServer, "")
	t.Setenv(config.EnvFallbackSettings, filepath.Join(home, "missing-fallback.toml"))
	server := testutil.SplitNightlyMockServer(t)
	settings := config.DefaultSettings()
	settings.DistServer = server.URL + "/corp/cjv"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))
	return home, "nightly-1.2.0-alpha.20260822010101"
}

func TestInstallWithComponentsEmitsSingleJSON(t *testing.T) {
	home, name := setupComponentOutputTest(t)
	app := newApplication("dev", "")
	stdout, err := captureStdout(t, func() error {
		return app.execute([]string{"--json", "install", "nightly", "--component", "docs"})
	})
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(home, "docs", name, "main", "index.html"))
	var result installResult
	require.NoError(t, json.Unmarshal([]byte(stdout), &result), stdout)
	require.Equal(t, []string{"docs"}, result.Components)

	// Idempotent SDK installation has the same success semantics in JSON mode.
	app = newApplication("dev", "")
	stdout, err = captureStdout(t, func() error { return app.execute([]string{"--json", "install", "nightly"}) })
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(stdout)), stdout)
}

func TestComponentAddJSONIncludesResultOnInstallAndSkip(t *testing.T) {
	home, name := setupComponentOutputTest(t)
	require.NoError(t, lifecycle.Install(t.Context(), lifecycle.InstallRequest{Toolchain: "nightly"}, lifecycle.Options{}))
	for _, force := range []bool{false, false, true} {
		app := newApplication("dev", "")
		args := []string{"--json", "component", "add", "docs", "--toolchain", name}
		if force {
			args = append(args, "--force")
		}
		stdout, err := captureStdout(t, func() error { return app.execute(args) })
		require.NoError(t, err)
		var result componentAddResult
		require.NoError(t, json.Unmarshal([]byte(stdout), &result), stdout)
		require.Equal(t, name, result.Toolchain)
		require.Equal(t, force, result.Forced)
		require.Len(t, result.Components, 1)
		require.Equal(t, "docs", string(result.Components[0]))
		require.FileExists(t, filepath.Join(home, "docs", name, "main", "index.html"))
	}
	app := newApplication("dev", "")
	stdout, err := captureStdout(t, func() error {
		return app.execute([]string{"--json", "component", "add", "not-a-component", "--toolchain", name})
	})
	require.Error(t, err)
	require.True(t, json.Valid([]byte(stdout)), stdout)
}

func TestComponentProgressUsesCommandWriterAndQuietInstaller(t *testing.T) {
	_, name := setupComponentOutputTest(t)
	require.NoError(t, lifecycle.Install(t.Context(), lifecycle.InstallRequest{Toolchain: "nightly"}, lifecycle.Options{}))
	// Proxy auto-install remains silent even if a caller supplies a reporter.
	stdout, err := captureStdout(t, func() error {
		return lifecycle.InstallComponentsForToolchain(t.Context(), name, []string{"docs"}, lifecycle.Options{
			Report: func(string, i18n.MsgData) { t.Fatal("quiet component install emitted progress") },
		})
	})
	require.NoError(t, err)
	require.Empty(t, stdout)

	app := newApplication("dev", "")
	var commandOutput bytes.Buffer
	app.rootCmd.SetOut(&commandOutput)
	stdout, err = captureStdout(t, func() error {
		return app.execute([]string{"component", "add", "docs", "--toolchain", name, "--force"})
	})
	require.NoError(t, err)
	require.Empty(t, stdout, "progress bypassed the command writer")
	require.Contains(t, commandOutput.String(), "docs")
	require.Contains(t, commandOutput.String(), name)
	// The test changes neither the process's real home nor its configured PATH.
	require.Equal(t, "1", os.Getenv(config.EnvNoPathSetup))
}
