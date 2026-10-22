package showcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/cli/clitest"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// checkHome isolates a home whose settings use manifestURL (or distServer)
// and creates the named toolchain directories without installing them.
func checkHome(t *testing.T, settings config.Settings, dirs ...string) string {
	t.Helper()
	home := t.TempDir()
	configtest.Isolate(t, home)
	require.NoError(t, config.EnsureDirs())
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))
	for _, dir := range dirs {
		require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", dir), 0o755))
	}
	return home
}

func manifestSettings(serverURL string) config.Settings {
	settings := config.DefaultSettings()
	settings.ManifestURL = serverURL + "/sdk-versions.json"
	return settings
}

// runCheckJSON runs `cjv check --json` and decodes its result.
func runCheckJSON(t *testing.T, c *commands) checkResult {
	t.Helper()
	c.ctx.Output.SetJSONMode(true)
	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetOut(&buf)
	require.NoError(t, c.runCheck(cmd, nil))
	var got checkResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got), buf.String())
	return got
}

func entryNames(entries []checkEntry) []string {
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name)
	}
	return names
}

func TestRunCheckNoToolchains(t *testing.T) {
	configtest.Isolate(t, t.TempDir())
	got := runCheckJSON(t, newTestCommands())
	assert.True(t, got.NoneInstalled, "no toolchains is a no-op, not an error")
	assert.Empty(t, got.Toolchains)
}

func TestRunCheckTrackedChannelsUpToDate(t *testing.T) {
	c := newTestCommands()
	checkHome(t, manifestSettings(testutil.ValidMockServer(t).URL))
	clitest.InstallToolchain(t, c.ctx, "lts")
	clitest.InstallToolchain(t, c.ctx, "sts")

	got := runCheckJSON(t, c)
	require.ElementsMatch(t, []string{"lts", "sts"}, entryNames(got.Toolchains))
	for _, e := range got.Toolchains {
		assert.NotEmpty(t, e.Latest, e.Name)
		assert.False(t, e.UpdateAvailable, e.Name)
		assert.Empty(t, e.Error, e.Name)
	}

	c.ctx.Output.SetJSONMode(false)
	out, err := clitest.CommandOutput(t, func(cmd *cobra.Command) error {
		cmd.SetContext(context.Background())
		return c.runCheck(cmd, nil)
	})
	require.NoError(t, err)
	assert.Contains(t, out, "lts ✓")
	assert.Contains(t, out, "sts ✓")
}

// Inspection keeps legacy fixed directories pinned and does not create
// tracking installations as a side effect.
func TestRunCheckDoesNotMigrateLegacyChannels(t *testing.T) {
	home := checkHome(t, manifestSettings(testutil.ValidMockServer(t).URL), "lts-1.0.0", "sts-1.0.0", "my-custom-sdk")

	got := runCheckJSON(t, newTestCommands())
	assert.False(t, got.HasUpdates)
	byName := map[string]checkEntry{}
	for _, e := range got.Toolchains {
		byName[e.Name] = e
	}
	require.ElementsMatch(t, []string{"lts-1.0.0", "sts-1.0.0"}, entryNames(got.Toolchains))
	for _, name := range []string{"lts-1.0.0", "sts-1.0.0"} {
		assert.Equal(t, "pinned", byName[name].Status)
		assert.Equal(t, name, byName[name].Selected)
	}
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "lts"))
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "sts"))
}

func TestRunCheckNightlyUsesUnifiedDistServer(t *testing.T) {
	c := newTestCommands()
	settings := config.DefaultSettings()
	settings.DistServer = testutil.SplitNightlyMockServer(t).URL + "/corp/cjv"
	checkHome(t, settings)
	clitest.InstallToolchain(t, c.ctx, "nightly")

	got := runCheckJSON(t, c)
	require.Len(t, got.Toolchains, 1)
	assert.Empty(t, got.Toolchains[0].Error)
	assert.Equal(t, "nightly-1.2.0-alpha.20260822010101", got.Toolchains[0].Latest)
}

func TestRunCheckLeavesFixedTargetsPinned(t *testing.T) {
	server := testutil.ManifestOnlyServer(t, testutil.ManifestWithPlatformGap())
	checkHome(t, manifestSettings(server.URL), "lts-1.0.0-linux-x64-ohos")

	got := runCheckJSON(t, newTestCommands())
	require.Len(t, got.Toolchains, 1)
	assert.Equal(t, "lts-1.0.0-linux-x64-ohos", got.Toolchains[0].Selected)
	assert.Equal(t, "pinned", got.Toolchains[0].Status)
	assert.False(t, got.Toolchains[0].UpdateAvailable)
	assert.False(t, got.Toolchains[0].NotForTarget)
}
