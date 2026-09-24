package selfmgmt

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/output"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelfUpdateJSONReportsSkippedBuildWithoutLeakingText(t *testing.T) {
	for _, tc := range []struct {
		name, version, updateURL, status string
	}{
		{"unconfigured", "1.0.0", "", "skipped"},
		{"development", "dev", "https://example.invalid/owner/repo/releases", "dev"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config.IsolateForTest(t, t.TempDir())
			t.Setenv(config.EnvNoPathSetup, "1")
			renderer := &output.Renderer{JSON: true}

			// Capture process stdout as well as Cobra's writer: implementations
			// that print around the renderer corrupt the actual CLI JSON stream.
			leaked, err := os.CreateTemp(t.TempDir(), "stdout-*")
			require.NoError(t, err)
			previousStdout := os.Stdout
			os.Stdout = leaked
			t.Cleanup(func() {
				os.Stdout = previousStdout
				_ = leaked.Close()
			})
			var stdout bytes.Buffer
			cmd := NewSelfCommand(tc.version, tc.updateURL, renderer)
			cmd.SetOut(&stdout)
			cmd.SetArgs([]string{"update"})
			require.NoError(t, cmd.Execute())
			data, err := os.ReadFile(leaked.Name())
			require.NoError(t, err)
			assert.Empty(t, string(data), "all output must use the command renderer")
			var result struct {
				Version string `json:"version"`
				Updated bool   `json:"updated"`
				Status  string `json:"status"`
			}
			require.NoError(t, json.Unmarshal(stdout.Bytes(), &result))
			assert.Equal(t, tc.version, result.Version)
			assert.False(t, result.Updated)
			assert.Equal(t, tc.status, result.Status)
		})
	}
}

func TestNewSelfCommandWiresSubcommandsAndUpdate(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)

	cmd := NewSelfCommand("dev", "", &output.Renderer{})

	assert.NotNil(t, cmd)
	assert.NotNil(t, findSubcommand(cmd, "update"))
	assert.NotNil(t, findSubcommand(cmd, "uninstall"))

	update := findSubcommand(cmd, "update")
	require.NoError(t, update.RunE(update, nil))
	assert.FileExists(t, filepath.Join(home, "bin", sdktools.CjvBinaryName()))
	assert.FileExists(t, filepath.Join(home, "bin", sdktools.PlatformBinaryName("cjc")))
}

func TestUpdateManagedRegeneratesEnvScripts(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)

	scripts := []string{"env"}
	if runtime.GOOS == "windows" {
		scripts = []string{"env.ps1", "env.bat"}
	}
	for _, name := range scripts {
		require.NoError(t, os.WriteFile(filepath.Join(home, name), []byte("outdated script"), 0o644))
	}
	result, err := UpdateManaged(context.Background(), "", "dev", nil)
	require.NoError(t, err)
	assert.False(t, result.Updated)
	for _, name := range scripts {
		data, err := os.ReadFile(filepath.Join(home, name))
		require.NoError(t, err)
		assert.NotEmpty(t, data)
		assert.NotEqual(t, "outdated script", string(data))
	}
}

func TestSelfUninstallDoesNotCleanPathWhenRemoveHomeFails(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)

	oldEnsure := ensureSelfManagedExecutable
	oldRemove := removeSelfHomeDir
	oldCleanup := cleanupSelfPathEntries
	ensureSelfManagedExecutable = func() (string, error) {
		return filepath.Join(home, "bin", sdktools.CjvBinaryName()), nil
	}
	removeSelfHomeDir = func(home, managedExe string) error {
		return errors.New("remove failed")
	}
	cleanupCalled := false
	cleanupSelfPathEntries = func() {
		cleanupCalled = true
	}
	t.Cleanup(func() {
		ensureSelfManagedExecutable = oldEnsure
		removeSelfHomeDir = oldRemove
		cleanupSelfPathEntries = oldCleanup
	})

	cmd := NewSelfCommand("dev", "", &output.Renderer{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.SetArgs([]string{"uninstall", "--yes"})
	require.Error(t, cmd.Execute())
	assert.False(t, cleanupCalled)
}

func TestSelfCommandsKeepUninstallConfirmationPerInvocation(t *testing.T) {
	config.IsolateForTest(t, t.TempDir())
	oldEnsure := ensureSelfManagedExecutable
	stopBeforeRemoval := errors.New("stop before removal")
	ensureSelfManagedExecutable = func() (string, error) { return "", stopBeforeRemoval }
	t.Cleanup(func() { ensureSelfManagedExecutable = oldEnsure })

	first := NewSelfCommand("dev", "", &output.Renderer{JSON: true})
	second := NewSelfCommand("dev", "", &output.Renderer{JSON: true})
	for _, cmd := range []*cobra.Command{first, second} {
		cmd.SetOut(io.Discard)
		cmd.SetErr(io.Discard)
		cmd.SilenceErrors = true
		cmd.SilenceUsage = true
	}
	first.SetArgs([]string{"uninstall", "--yes"})
	require.ErrorIs(t, first.Execute(), stopBeforeRemoval)
	second.SetArgs([]string{"uninstall"})
	var unsupported *cjverr.UnsupportedForJSONError
	require.ErrorAs(t, second.Execute(), &unsupported)
}

func findSubcommand(cmd *cobra.Command, name string) *cobra.Command {
	for _, child := range cmd.Commands() {
		if child.Name() == name {
			return child
		}
	}
	return nil
}
