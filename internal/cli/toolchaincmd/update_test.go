package toolchaincmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/cli/clitest"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for the update command: argument handling, the self-update decision
// and result rendering. Update semantics live with lifecycle.UpdateInstalled
// and UpdateAll.

// runUpdateJSON runs update with args in JSON mode and decodes the result.
func runUpdateJSON(t *testing.T, c *update, args ...string) updateResult {
	t.Helper()
	c.ctx.Output.SetJSONMode(true)
	var out bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetOut(&out)
	require.NoError(t, c.run(cmd, args))
	var result updateResult
	require.NoError(t, json.Unmarshal(out.Bytes(), &result), out.String())
	return result
}

func TestRunUpdateWithNoToolchainsIsNoOp(t *testing.T) {
	configtest.Isolate(t, t.TempDir())
	result := runUpdateJSON(t, newUpdate(newTestContext()))
	assert.True(t, result.NoneInstalled)
	assert.Empty(t, result.Updates)
}

func TestRunUpdateLatestToolchainHasNoUpdates(t *testing.T) {
	for _, args := range [][]string{nil, {"lts"}, {"lts-1.0.5"}} {
		t.Run(fmt.Sprint(args), func(t *testing.T) {
			validManifestHome(t)
			c := newUpdate(newTestContext())
			c.noSelfUpdate = true
			clitest.InstallToolchain(t, c.ctx, "lts")
			result := runUpdateJSON(t, c, args...)
			assert.Empty(t, result.Updates)
			assert.False(t, result.NoneInstalled)
		})
	}
}

func TestRunUpdateRunsSelfCheckWhenSettingsAvailable(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	require.NoError(t, config.EnsureDirs())
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "local-sdk"), 0o755))
	settings := config.DefaultSettings()
	settings.AutoSelfUpdate = config.AutoSelfUpdateCheck
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	require.NoError(t, newUpdate(newTestContext()).run(cmd, nil))
}

func TestRunUpdateRejectsInvalidAndUnknownNames(t *testing.T) {
	configtest.Isolate(t, t.TempDir())
	c := newUpdate(newTestContext())
	cmd := &cobra.Command{}
	require.Error(t, c.run(cmd, []string{"+bad"}))
	require.Error(t, c.run(cmd, []string{"local-sdk"}))
	require.Error(t, c.run(cmd, []string{"nonexistent-99.99"}))
}

func TestRunUpdateRendersJSONResult(t *testing.T) {
	home := validManifestHome(t)
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.0"), 0o755))
	c := newUpdate(newTestContext())
	c.noSelfUpdate = true

	result := runUpdateJSON(t, c)
	assert.Equal(t, []updateEntry{{From: "lts", To: "lts-1.0.5"}}, result.Updates)
	assert.False(t, result.NoneInstalled)
}

func TestUpdateExistingLatestAcrossRenderModes(t *testing.T) {
	for _, jsonMode := range []bool{false, true} {
		t.Run(fmt.Sprint(jsonMode), func(t *testing.T) {
			c := newUpdate(newTestContext())
			c.ctx.Output.SetJSONMode(jsonMode)
			home := validManifestHome(t)
			t.Setenv(config.EnvNoPathSetup, "1")
			for _, name := range []string{"lts-1.0.0", "lts-1.0.5"} {
				require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", name), 0o755))
			}
			settingsPath := filepath.Join(home, ".cjv", "settings.toml")
			settings, err := config.LoadSettings(settingsPath)
			require.NoError(t, err)
			settings.DefaultToolchain = "lts"
			settings.Overrides[filepath.Join(home, "project")] = "lts"
			require.NoError(t, config.SaveSettings(settings, settingsPath))

			cmd := &cobra.Command{}
			cmd.SetContext(context.Background())
			require.NoError(t, c.run(cmd, nil), "render mode must not decide whether update succeeds")
			loaded, err := config.LoadSettings(settingsPath)
			require.NoError(t, err)
			require.Equal(t, "lts", loaded.DefaultToolchain)
			require.DirExists(t, filepath.Join(home, "toolchains", "lts-1.0.0"), "legacy fixed installation survives the channel update")
		})
	}
}

func TestUpdateCancellationWhileHomeLocked(t *testing.T) {
	for _, args := range [][]string{
		{"update", "sts"},
		{"update", "--no-self-update"},
		{"update"},
	} {
		t.Run(fmt.Sprint(args), func(t *testing.T) {
			home := t.TempDir()
			configtest.Isolate(t, home)
			t.Setenv(config.EnvNoPathSetup, "1")
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests.Add(1)
				http.NotFound(w, r)
			}))
			t.Cleanup(server.Close)
			sf, err := config.DefaultSettingsFile()
			require.NoError(t, err)
			manifest := server.URL + "/manifest.json"
			auto := config.AutoSelfUpdateEnable
			_, err = sf.Update(config.SettingsUpdate{ManifestURL: &manifest, AutoSelfUpdate: &auto})
			require.NoError(t, err)
			lock, err := toolchain.LockHome(t.Context())
			require.NoError(t, err)
			var releaseOnce sync.Once
			release := func() { releaseOnce.Do(func() { _ = lock.Close() }) }
			t.Cleanup(release)
			// A regression must fail promptly rather than hanging the test suite
			// forever inside recovery with a background context.
			fallback := time.AfterFunc(2*time.Second, release)
			defer fallback.Stop()
			ctx, cancel := context.WithTimeout(t.Context(), 75*time.Millisecond)
			defer cancel()
			cli := clitest.Root(Register)
			cli.Version, cli.UpdateURL = "1.0.0", server.URL+"/self-update"
			cli.Root.SetContext(ctx)
			start := time.Now()
			_, err = clitest.Execute(t, cli, args...)
			require.ErrorIs(t, err, context.DeadlineExceeded)
			assert.Less(t, time.Since(start), time.Second, "cancellation must not wait for another operation's home lock")
			assert.Zero(t, requests.Load(), "canceled updates must not start distribution or self-update requests")
			assert.NoDirExists(t, filepath.Join(home, "bin"), "auto-self-update must not bootstrap a managed executable after cancellation")
		})
	}
}

func TestUpdateEntriesKeepsAppliedOutcomesOnly(t *testing.T) {
	entries := updateEntries([]lifecycle.UpdateOutcome{
		{Name: "local-sdk", Status: lifecycle.UpdateSkipped},
		{Name: "lts-1.0.5", Replacement: "lts-1.0.5", Status: lifecycle.UpdateUpToDate},
		{Name: "sts-1.0.0", Replacement: "sts-2.0.0", Status: lifecycle.UpdateApplied},
		{Name: "nightly-1", Status: lifecycle.UpdateFailed},
	})
	assert.Equal(t, []updateEntry{{From: "sts-1.0.0", To: "sts-2.0.0"}}, entries)
	assert.Nil(t, updateEntries(nil))
}
