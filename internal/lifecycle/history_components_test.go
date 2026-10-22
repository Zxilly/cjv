package lifecycle_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/require"
)

func TestHistoryRestoresDocsAndMaterializesLinkedStdx(t *testing.T) {
	f := newUpgradeFixture(t, false, false)
	channel := fixtureChannel(t, f)
	source := linkUpgradeStdx(t, channel, "saved user library")
	_, err := lifecycle.UpdateInstalled(t.Context(), toolchain.ToolchainName{Channel: toolchain.LTS}, lifecycle.Options{})
	require.NoError(t, err)
	entries, err := lifecycle.ListHistory(t.Context(), "lts")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Len(t, entries[0].Members[0].Roots, 3)
	// Changes to either the linked source or the live SDK do not mutate history.
	require.NoError(t, os.WriteFile(filepath.Join(source, "dynamic", "user.txt"), []byte("changed source"), 0o644))
	require.NoError(t, os.WriteFile(compilerPath(channel.TcDir), []byte("changed live compiler"), 0o755))
	_, err = lifecycle.RollbackHistory(t.Context(), "lts", entries[0].ID, lifecycle.Options{})
	require.NoError(t, err)
	for _, path := range []string{filepath.Join(channel.DocsDir, "main", "index.html"), filepath.Join(channel.DocsDir, "stdx", "index.html")} {
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "1.0.0", string(data))
	}
	data, err := os.ReadFile(filepath.Join(channel.StdxDir, "dynamic", "user.txt"))
	require.NoError(t, err)
	require.Equal(t, "saved user library", string(data))
	intents, err := component.InstalledIntents(channel)
	require.NoError(t, err)
	require.Contains(t, intents, component.Intent{Name: component.Stdx})
	require.NoError(t, component.Remove(channel, component.Stdx))
	require.NoFileExists(t, filepath.Join(channel.StdxDir, "dynamic", "user.txt"))
	data, err = os.ReadFile(filepath.Join(source, "dynamic", "user.txt"))
	require.NoError(t, err)
	require.Equal(t, "changed source", string(data))
	require.FileExists(t, filepath.Join(f.oldRoots.StdxDir, "dynamic", "version.txt"))
}

func TestReinstallRepairsCompilerAndRestoresComponentsWithoutAdvancingRelease(t *testing.T) {
	f := newUpgradeFixture(t, false, false)
	channel := fixtureChannel(t, f)
	before, err := f.sf.Load()
	require.NoError(t, err)
	require.NoError(t, os.Remove(compilerPath(channel.TcDir)))
	require.NoError(t, lifecycle.Install(t.Context(), lifecycle.InstallRequest{Toolchain: "lts", Reinstall: true}, lifecycle.Options{}))
	require.FileExists(t, compilerPath(channel.TcDir))
	record, err := toolchain.ReadInstallation(channel.TcDir)
	require.NoError(t, err)
	require.Equal(t, "lts-1.0.0", record.Release)
	for _, path := range []string{filepath.Join(channel.StdxDir, "dynamic", "version.txt"), filepath.Join(channel.DocsDir, "main", "index.html"), filepath.Join(channel.DocsDir, "stdx", "index.html")} {
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		require.Equal(t, "1.0.0", string(data))
	}
	after, err := f.sf.Load()
	require.NoError(t, err)
	require.Equal(t, before.DefaultToolchain, after.DefaultToolchain)
	require.Equal(t, before.Overrides, after.Overrides)
	history, err := lifecycle.ListHistory(t.Context(), "lts")
	require.NoError(t, err)
	require.Empty(t, history)
}
