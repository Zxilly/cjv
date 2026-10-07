package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/testutil"
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

func TestRunPreparationRejectsInvalidSelectionsWithoutInstalling(t *testing.T) {
	for _, install := range []bool{false, true} {
		for _, selection := range []struct {
			name    string
			missing bool
			message string
		}{
			{name: "bad/name"},
			{name: "sts-2.0.0-win32-x64-ohos", message: "target variant"},
			{name: "missing-custom-sdk", missing: true},
		} {
			t.Run(fmt.Sprintf("%s/install=%t", selection.name, install), func(t *testing.T) {
				home := t.TempDir()
				config.IsolateForTest(t, home)
				var requests atomic.Int32
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					requests.Add(1)
					http.NotFound(w, r)
				}))
				t.Cleanup(server.Close)
				sf, err := config.DefaultSettingsFile()
				require.NoError(t, err)
				url := server.URL + "/sdk-versions.json"
				_, err = sf.Update(config.SettingsUpdate{ManifestURL: &url})
				require.NoError(t, err)
				args := []string{"run"}
				if install {
					args = append(args, "--install")
				}
				args = append(args, selection.name, "go", "version")
				_, err = executeWithOutput(t, newApplication("dev", ""), args)
				require.Error(t, err)
				var missing *cjverr.ToolchainNotInstalledError
				assert.Equal(t, selection.missing, errors.As(err, &missing), "validation errors must retain their type")
				if selection.message != "" {
					assert.Contains(t, err.Error(), selection.message)
				}
				assert.Zero(t, requests.Load(), "an invalid or custom selection must never request a manifest")
			})
		}
	}
}

func TestRunPreparationInstallsOnlyMissingOfficialToolchain(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	upstream := testutil.ValidMockServer(t)
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		upstream.Config.Handler.ServeHTTP(w, r)
	}))
	t.Cleanup(server.Close)
	sf, err := config.DefaultSettingsFile()
	require.NoError(t, err)
	url := server.URL + "/sdk-versions.json"
	_, err = sf.Update(config.SettingsUpdate{ManifestURL: &url})
	require.NoError(t, err)

	_, err = executeWithOutput(t, newApplication("dev", ""), []string{"run", "lts", "go", "version"})
	var missing *cjverr.ToolchainNotInstalledError
	require.ErrorAs(t, err, &missing)
	assert.Zero(t, requests.Load())

	_, err = executeWithOutput(t, newApplication("dev", ""), []string{"run", "--install", "lts", "go", "version"})
	require.NoError(t, err)
	require.Positive(t, requests.Load())
	require.DirExists(t, filepath.Join(home, "toolchains", "lts"))
	afterInstall := requests.Load()
	_, err = executeWithOutput(t, newApplication("dev", ""), []string{"run", "--install", "lts", "go", "version"})
	require.NoError(t, err)
	assert.Equal(t, afterInstall, requests.Load(), "an installed host does not need the distribution")
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
