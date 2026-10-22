package lifecycle

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/stretchr/testify/require"
)

type historyUpdateFixture struct {
	home, host, oldChecksum, newChecksum string
	republishTarget                      atomic.Bool
	publishDocs                          atomic.Bool
	requests                             atomic.Int64
}

func newHistoryUpdateFixture(t *testing.T) *historyUpdateFixture {
	t.Helper()
	f := &historyUpdateFixture{home: updateHome(t)}
	var err error
	f.host, err = target.CurrentHostTuple("")
	require.NoError(t, err)
	oldSDK, oldChecksum := testutil.MockTarGz(t, map[string]string{"cangjie/bin/cjc": "old compiler", "cangjie/bin/cjc.exe": "old compiler"})
	newSDK, newChecksum := testutil.MockTarGz(t, map[string]string{"cangjie/bin/cjc": "new compiler", "cangjie/bin/cjc.exe": "new compiler"})
	docs, docsChecksum := testutil.MockTarGz(t, map[string]string{"index.html": "documentation"})
	f.oldChecksum, f.newChecksum = oldChecksum, newChecksum
	f.publishDocs.Store(true)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f.requests.Add(1)
		switch r.URL.Path {
		case "/old.tar.gz":
			_, _ = w.Write(oldSDK)
		case "/new.tar.gz":
			_, _ = w.Write(newSDK)
		case "/docs.tar.gz":
			_, _ = w.Write(docs)
		case "/versions.json":
			base := "http://" + r.Host
			original := dist.DownloadInfo{Name: "old.tar.gz", URL: base + "/old.tar.gz", SHA256: oldChecksum}
			updated := original
			if f.republishTarget.Load() {
				updated = dist.DownloadInfo{Name: "new.tar.gz", URL: base + "/new.tar.gz", SHA256: newChecksum}
			}
			channel := dist.ChannelInfo{Latest: "1.0.0", Versions: map[string]map[string]dist.DownloadInfo{"1.0.0": {f.host: original, f.host + "-ohos": updated}}}
			if f.publishDocs.Load() {
				channel.Components = map[string]dist.ComponentSet{"1.0.0": {Docs: &dist.ComponentInfo{Name: "docs.tar.gz", URL: base + "/docs.tar.gz", SHA256: docsChecksum}}}
			}
			var manifest dist.Manifest
			manifest.Channels.STS, manifest.Channels.LTS = channel, channel
			_ = json.NewEncoder(w).Encode(manifest)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	sf, err := config.DefaultSettingsFile()
	require.NoError(t, err)
	url := server.URL + "/versions.json"
	_, err = sf.Update(config.SettingsUpdate{ManifestURL: &url})
	require.NoError(t, err)
	return f
}

func TestHistoryRetainsTargetOnlyUpdateAndRollsBackWholeGroup(t *testing.T) {
	f := newHistoryUpdateFixture(t)
	for _, req := range []InstallRequest{
		{Toolchain: "sts"},
		{Toolchain: "sts", Targets: []string{"ohos"}},
		{Toolchain: "sts", Components: []string{"docs"}},
		{Toolchain: "sts", Reinstall: true},
	} {
		require.NoError(t, Install(t.Context(), req, Options{}))
		entries, err := ListHistory(t.Context(), "sts")
		require.NoError(t, err)
		require.Empty(t, entries, "first installs, additions and repairs do not create update history")
	}
	hostBefore := readRelease(t, f.home, "sts")
	targetName := "sts-" + f.host + "-ohos"
	targetBefore := readRelease(t, f.home, targetName)
	f.republishTarget.Store(true)
	outcome, err := UpdateInstalled(t.Context(), parse(t, "sts"), Options{})
	require.NoError(t, err)
	require.Equal(t, UpdateApplied, outcome.Status)
	require.Equal(t, hostBefore, readRelease(t, f.home, "sts"))
	require.Equal(t, f.newChecksum, readRelease(t, f.home, targetName).SHA256)
	entries, err := ListHistory(t.Context(), "sts")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Len(t, entries[0].Members, 2)
	require.Equal(t, "sts", entries[0].Members[0].Name)
	require.Equal(t, hostBefore, entries[0].Members[0].installation())
	require.Equal(t, targetBefore, entries[0].Members[1].installation())
	requests := f.requests.Load()
	_, err = RollbackHistory(dist.WithOffline(t.Context(), true), "sts", entries[0].ID, Options{})
	require.NoError(t, err)
	require.Equal(t, requests, f.requests.Load(), "rollback must not fetch the overwritten target artifact")
	for _, name := range []string{"sts", targetName} {
		require.Equal(t, f.oldChecksum, readRelease(t, f.home, name).SHA256)
		compiler, err := os.ReadFile(filepath.Join(f.home, "toolchains", name, "bin", "cjc"))
		require.NoError(t, err)
		require.Equal(t, "old compiler", string(compiler))
		docs, err := os.ReadFile(filepath.Join(f.home, "docs", name, "main", "index.html"))
		require.NoError(t, err)
		require.Equal(t, "documentation", string(docs))
	}
}

func TestReinstallForceDropsUnavailableComponentButUpdatePreservesIt(t *testing.T) {
	f := newHistoryUpdateFixture(t)
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts", Components: []string{"docs"}}, Options{}))
	f.publishDocs.Store(false)
	outcome, err := UpdateInstalled(t.Context(), parse(t, "sts"), Options{})
	require.NoError(t, err, "an unchanged SDK keeps its already installed components without downloading them")
	require.Equal(t, UpdateUpToDate, outcome.Status)
	roots, err := component.RootsFor("sts")
	require.NoError(t, err)
	docsPath := filepath.Join(roots.DocsDir, "main", "index.html")
	require.FileExists(t, docsPath)
	compiler, err := sdktools.ResolveInstalledToolBinaryForTuple(roots.TcDir, "cjc", f.host)
	require.NoError(t, err)
	require.NoError(t, os.Remove(compiler))
	err = Install(t.Context(), InstallRequest{Toolchain: "sts", Reinstall: true}, Options{})
	require.Error(t, err, "repair without force must preserve the unavailable component and reject the replacement")
	require.FileExists(t, docsPath)
	require.NoFileExists(t, compiler)
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts", Reinstall: true, Force: true}, Options{}))
	require.FileExists(t, compiler)
	require.NoFileExists(t, docsPath)
	require.False(t, component.IsInstalled(roots.TcDir, component.Docs))
	require.Equal(t, "sts-1.0.0", readRelease(t, f.home, "sts").Release)
	entries, err := ListHistory(t.Context(), "sts")
	require.NoError(t, err)
	require.Empty(t, entries)
}
