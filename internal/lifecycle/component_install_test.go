package lifecycle

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallComponentsListRestoresBatchAfterLaterArchiveFailure(t *testing.T) {
	const toolchainName = "lts-1.0.5"
	config.IsolateForTest(t, t.TempDir())
	roots, err := component.RootsFor(toolchainName)
	require.NoError(t, err)
	docsRoot := filepath.Join(roots.DocsDir, "main")
	require.NoError(t, os.MkdirAll(docsRoot, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(docsRoot, "old.html"), []byte("old docs"), 0o644))
	require.NoError(t, component.WriteManifest(roots.TcDir, component.Docs, []string{"old.html"}))
	require.NoError(t, component.WriteManifest(roots.TcDir, component.Stdx, []string{"dynamic/existing.so"}))

	var archive bytes.Buffer
	w := zip.NewWriter(&archive)
	file, err := w.Create("new.html")
	require.NoError(t, err)
	_, err = file.Write([]byte("new docs"))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	sum := sha256.Sum256(archive.Bytes())
	checksum := hex.EncodeToString(sum[:])

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	manifest := dist.Manifest{}
	channel := dist.ChannelInfo{
		Latest: "1.0.5",
		Versions: map[string]map[string]dist.DownloadInfo{"1.0.5": {
			"linux-x64": {Name: "sdk.zip", URL: server.URL + "/sdk.zip", SHA256: checksum},
		}},
	}
	manifest.Channels.LTS = channel
	manifest.Channels.STS = channel
	manifest.Channels.LTS.Components = map[string]dist.ComponentSet{"1.0.5": {
		Docs:     &dist.ComponentInfo{URL: server.URL + "/docs.zip", SHA256: checksum},
		StdxDocs: &dist.ComponentInfo{URL: server.URL + "/stdx-docs.zip"},
	}}
	mux.HandleFunc("/versions.json", func(w http.ResponseWriter, _ *http.Request) {
		assert.NoError(t, json.NewEncoder(w).Encode(manifest))
	})
	mux.HandleFunc("/docs.zip", func(w http.ResponseWriter, _ *http.Request) {
		_, err := w.Write(archive.Bytes())
		assert.NoError(t, err)
	})
	var secondArchiveRequested atomic.Bool
	mux.HandleFunc("/stdx-docs.zip", func(w http.ResponseWriter, _ *http.Request) {
		secondArchiveRequested.Store(true)
		// Prove the first installation really committed before the next
		// archive failed; the final assertions therefore exercise batch recovery.
		assert.FileExists(t, filepath.Join(docsRoot, "new.html"))
		assert.NoFileExists(t, filepath.Join(docsRoot, "old.html"))
		_, err := w.Write([]byte("invalid archive"))
		assert.NoError(t, err)
	})
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/versions.json"
	sf, err := config.DefaultSettingsFile()
	require.NoError(t, err)
	require.NoError(t, sf.Save(&settings))

	err = InstallComponentsList(context.Background(), toolchainName, []string{"docs", "stdx-docs"}, true, true, nil, quietLifecycleOptions())
	require.Error(t, err)
	assert.True(t, secondArchiveRequested.Load())
	data, readErr := os.ReadFile(filepath.Join(docsRoot, "old.html"))
	require.NoError(t, readErr)
	assert.Equal(t, "old docs", string(data))
	assert.NoFileExists(t, filepath.Join(docsRoot, "new.html"))
	assert.NoDirExists(t, filepath.Join(roots.DocsDir, "stdx"))
	installed, readErr := component.ListInstalled(roots.TcDir)
	require.NoError(t, readErr)
	assert.ElementsMatch(t, []component.Name{component.Docs, component.Stdx}, installed)
}
