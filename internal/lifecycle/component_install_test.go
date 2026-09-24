package lifecycle

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func quietLifecycleOptions() Options {
	return Options{}
}

// installedToolchainHome isolates CJV_HOME with an (empty) installed
// toolchain directory named tcName, and returns that directory.
func installedToolchainHome(t *testing.T, tcName string) string {
	t.Helper()
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv(config.EnvDistServer, "")
	tcDir := filepath.Join(home, "toolchains", tcName)
	require.NoError(t, os.MkdirAll(tcDir, 0o755))
	return tcDir
}

// fakeComponentInstall replaces the real component installer with install,
// which sees the roots, component, tuple and force flag the batch resolved.
func fakeComponentInstall(install func(roots component.Roots, name component.Name, tuple string, force bool) error) Options {
	return Options{ComponentInstall: func(_ context.Context, roots component.Roots, _ toolchain.ToolchainName, name component.Name, tuple, _ string, force bool) error {
		return install(roots, name, tuple, force)
	}}
}

func TestInstallComponentsInputValidationAndAlreadyInstalled(t *testing.T) {
	tcName := "lts-1.0.5"
	tcDir := installedToolchainHome(t, tcName)
	require.Error(t, InstallComponents(context.Background(), "+bad", []string{"docs"}, false, quietLifecycleOptions()))

	err := InstallComponents(context.Background(), "local-sdk", []string{"docs"}, false, quietLifecycleOptions())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docs")

	require.Error(t, InstallComponents(context.Background(), tcName, []string{"unknown"}, false, quietLifecycleOptions()))

	var reported []string
	opts := fakeComponentInstall(func(roots component.Roots, name component.Name, _ string, _ bool) error {
		return &cjverr.ComponentAlreadyInstalledError{Toolchain: filepath.Base(roots.TcDir), Component: string(name)}
	})
	opts.Report = func(message string, _ i18n.MsgData) { reported = append(reported, message) }
	require.NoError(t, InstallComponents(context.Background(), tcName, []string{"docs"}, false, opts))
	assert.Equal(t, []string{"ComponentAlreadyInstalled"}, reported)
	assert.False(t, component.IsInstalled(tcDir, component.Docs))
}

func TestInstallComponentsRollsBackPreviousComponentOnLaterFailure(t *testing.T) {
	tcName := "lts-1.0.5"
	tcDir := installedToolchainHome(t, tcName)
	opts := fakeComponentInstall(func(roots component.Roots, name component.Name, _ string, _ bool) error {
		if name == component.Docs {
			return errors.New("docs failed")
		}
		require.NoError(t, os.MkdirAll(filepath.Join(roots.StdxDir, "dynamic"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(roots.StdxDir, "dynamic", "libfoo.so"), []byte("x"), 0o644))
		return component.WriteManifest(roots.TcDir, name, []string{"dynamic/libfoo.so"})
	})

	err := InstallComponents(context.Background(), tcName, []string{"stdx", "docs"}, false, opts)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "docs failed")
	assert.False(t, component.IsInstalled(tcDir, component.Stdx))
	stdxDir, dirErr := config.StdxDirFor(tcName)
	require.NoError(t, dirErr)
	assert.NoFileExists(t, filepath.Join(stdxDir, "dynamic", "libfoo.so"))
}

func TestInstallComponentsUsesTargetTupleForTargetVariant(t *testing.T) {
	const targetName = "lts-1.0.5-linux-x64-ohos"
	installedToolchainHome(t, targetName)
	var gotTuple, gotTcDir string
	opts := fakeComponentInstall(func(roots component.Roots, _ component.Name, tuple string, _ bool) error {
		gotTuple, gotTcDir = tuple, roots.TcDir
		return nil
	})

	require.NoError(t, InstallComponents(context.Background(), targetName, []string{"stdx"}, false, opts))

	// The target tuple encoded in the resolved name drives the stdx download,
	// not the host tuple.
	assert.Equal(t, "linux-x64-ohos", gotTuple)
	// Roots (and thus the manifest + StdxDir) are keyed by the full target name.
	assert.Equal(t, targetName, filepath.Base(gotTcDir))
}

func TestInstallComponentsForToolchainUsesInstalledToolchainQuietly(t *testing.T) {
	tcName := "lts-1.0.5"
	tcDir := installedToolchainHome(t, tcName)
	opts := fakeComponentInstall(func(roots component.Roots, name component.Name, _ string, _ bool) error {
		return component.WriteManifest(roots.TcDir, name, []string{"index.html"})
	})
	opts.Report = func(string, i18n.MsgData) { t.Fatal("quiet component install emitted progress") }

	require.NoError(t, InstallComponentsForToolchain(context.Background(), "lts", []string{"docs"}, opts))
	assert.True(t, component.IsInstalled(tcDir, component.Docs))

	require.NoError(t, InstallComponentsForToolchain(context.Background(), "lts", nil, opts), "no components is a no-op")
	require.Error(t, InstallComponentsForToolchain(context.Background(), "+bad", []string{"docs"}, opts))
	require.Error(t, InstallComponentsForToolchain(context.Background(), "sts-2.0.0", []string{"docs"}, opts), "missing toolchain")
}

func TestInstallComponentsRestoresBatchAfterLaterArchiveFailure(t *testing.T) {
	const toolchainName = "lts-1.0.5"
	config.IsolateForTest(t, t.TempDir())
	t.Setenv(config.EnvDistServer, "")
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

	err = InstallComponents(context.Background(), toolchainName, []string{"docs", "stdx-docs"}, true, quietLifecycleOptions())
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
