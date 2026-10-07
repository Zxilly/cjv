package lifecycle

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/progress"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/testutil"
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

// componentServer serves an LTS 1.0.5 manifest whose component set is
// built from archives: docs and stdx-docs from docsFiles, the stdx of every
// platform in stdxPlatforms from stdxFiles. The archive served for a path in
// broken is invalid instead. Settings are saved so OpenDistribution reads the
// server, and the paths requested are returned for assertions.
func componentServer(t *testing.T, docsFiles, stdxFiles map[string]string, stdxPlatforms []string, broken ...string) func() []string {
	t.Helper()
	docsData, docsSHA := testutil.MockTarGz(t, docsFiles)
	stdxData, stdxSHA := testutil.MockTarGz(t, stdxFiles)
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	set := dist.ComponentSet{
		Docs:     &dist.ComponentInfo{URL: server.URL + "/docs.tar.gz", SHA256: docsSHA},
		StdxDocs: &dist.ComponentInfo{URL: server.URL + "/stdx-docs.tar.gz", SHA256: docsSHA},
		Stdx:     map[string]dist.ComponentInfo{},
	}
	for _, platform := range stdxPlatforms {
		set.Stdx[platform] = dist.ComponentInfo{URL: server.URL + "/stdx/" + platform + ".tar.gz", SHA256: stdxSHA}
	}
	var manifest dist.Manifest
	channel := dist.ChannelInfo{
		Latest: "1.0.5",
		Versions: map[string]map[string]dist.DownloadInfo{"1.0.5": {
			"linux-x64": {Name: "sdk.zip", URL: server.URL + "/sdk.zip", SHA256: docsSHA},
		}},
	}
	manifest.Channels.LTS = channel
	manifest.Channels.STS = channel
	manifest.Channels.LTS.Components = map[string]dist.ComponentSet{"1.0.5": set}

	var mu sync.Mutex
	var requested []string
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requested = append(requested, r.URL.Path)
		mu.Unlock()
		switch {
		case r.URL.Path == "/versions.json":
			assert.NoError(t, json.NewEncoder(w).Encode(manifest))
		case slices.Contains(broken, r.URL.Path):
			_, _ = w.Write([]byte("invalid archive"))
		case strings.HasPrefix(r.URL.Path, "/stdx/"):
			_, _ = w.Write(stdxData)
		default:
			_, _ = w.Write(docsData)
		}
	})

	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/versions.json"
	sf, err := config.DefaultSettingsFile()
	require.NoError(t, err)
	require.NoError(t, sf.Save(&settings))
	return func() []string {
		mu.Lock()
		defer mu.Unlock()
		return append([]string(nil), requested...)
	}
}

// hostStdxPlatform is the stdx archive platform token of the test host.
func hostStdxPlatform(t *testing.T) string {
	t.Helper()
	tuple, err := sdktarget.CurrentHostTuple("")
	require.NoError(t, err)
	platform, err := sdktarget.StdxPlatformForTuple(tuple)
	require.NoError(t, err)
	return platform
}

func TestInstallComponentsInputValidationAndAlreadyInstalled(t *testing.T) {
	tcName := "lts-1.0.5"
	tcDir := installedToolchainHome(t, tcName)
	require.Error(t, InstallComponents(context.Background(), "+bad", []string{"docs"}, false, quietLifecycleOptions()))

	err := InstallComponents(context.Background(), "local-sdk", []string{"docs"}, false, quietLifecycleOptions())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docs")

	require.Error(t, InstallComponents(context.Background(), tcName, []string{"unknown"}, false, quietLifecycleOptions()))

	// An installed component is kept and reported without touching the
	// source: nothing is fetched and the manifest stays as it was.
	require.NoError(t, component.WriteManifest(tcDir, component.Docs, []string{"index.html"}))
	recorder := &testutil.ProgressRecorder{}
	require.NoError(t, InstallComponents(context.Background(), tcName, []string{"docs"}, false, Options{Progress: recorder}))
	assert.Equal(t, []progress.Kind{progress.ComponentAlreadyInstalled}, recorder.Kinds)
	assert.True(t, component.IsInstalled(tcDir, component.Docs))
}

func TestInstallComponentsRollsBackPreviousComponentOnLaterFailure(t *testing.T) {
	tcName := "lts-1.0.5"
	tcDir := installedToolchainHome(t, tcName)
	requested := componentServer(t,
		map[string]string{"index.html": "docs"},
		map[string]string{"top/dynamic/libfoo.so": "x"},
		[]string{hostStdxPlatform(t)},
		"/docs.tar.gz")

	err := InstallComponents(context.Background(), tcName, []string{"stdx", "docs"}, false, quietLifecycleOptions())

	require.Error(t, err)
	assert.Contains(t, requested(), "/docs.tar.gz", "docs failed only after stdx was installed")
	assert.False(t, component.IsInstalled(tcDir, component.Stdx))
	stdxDir, dirErr := config.StdxDirFor(tcName)
	require.NoError(t, dirErr)
	assert.NoFileExists(t, filepath.Join(stdxDir, "dynamic", "libfoo.so"))
}

func TestInstallComponentsUsesTargetTupleForTargetVariant(t *testing.T) {
	const targetName = "lts-1.0.5-linux-x64-ohos"
	installedToolchainHome(t, targetName)
	requested := componentServer(t,
		map[string]string{"index.html": "docs"},
		map[string]string{"top/dynamic/libfoo.so": "x"},
		[]string{hostStdxPlatform(t), "ohos-aarch64"})

	require.NoError(t, InstallComponents(context.Background(), targetName, []string{"stdx"}, false, quietLifecycleOptions()))

	// The target tuple encoded in the resolved name drives the stdx download,
	// not the host tuple.
	assert.Contains(t, requested(), "/stdx/ohos-aarch64.tar.gz")
	// Roots (and thus the manifest + StdxDir) are keyed by the full target name.
	stdxDir, err := config.StdxDirFor(targetName)
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(stdxDir, "dynamic", "libfoo.so"))
}

func TestInstallComponentsForToolchainResolvesInstalledToolchain(t *testing.T) {
	tcName := "lts-1.0.5"
	tcDir := installedToolchainHome(t, tcName)
	componentServer(t, map[string]string{"index.html": "docs"}, map[string]string{"top/dynamic/libfoo.so": "x"}, nil)

	// The channel name resolves to the installed version, and progress goes
	// to whatever sink the caller supplied.
	recorder := &testutil.ProgressRecorder{}
	require.NoError(t, InstallComponentsForToolchain(context.Background(), "lts", []string{"docs"}, Options{Progress: recorder}))
	channelDir, err := config.ToolchainDirFor("lts")
	require.NoError(t, err)
	assert.True(t, component.IsInstalled(channelDir, component.Docs))
	assert.False(t, component.IsInstalled(tcDir, component.Docs))
	assert.Contains(t, recorder.Kinds, progress.ComponentInstalled)

	require.NoError(t, InstallComponentsForToolchain(context.Background(), "lts", nil, quietLifecycleOptions()), "no components is a no-op")
	require.Error(t, InstallComponentsForToolchain(context.Background(), "+bad", []string{"docs"}, quietLifecycleOptions()))
	require.Error(t, InstallComponentsForToolchain(context.Background(), "sts-2.0.0", []string{"docs"}, quietLifecycleOptions()), "missing toolchain")
}

func TestInstallComponentsPreparesWholeBatchBeforePublishing(t *testing.T) {
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
	var docsRequests atomic.Int32
	mux.HandleFunc("/docs.zip", func(w http.ResponseWriter, _ *http.Request) {
		docsRequests.Add(1)
		_, err := w.Write(archive.Bytes())
		assert.NoError(t, err)
	})
	var secondArchiveRequested atomic.Bool
	var secondArchiveValid atomic.Bool
	mux.HandleFunc("/stdx-docs.zip", func(w http.ResponseWriter, _ *http.Request) {
		secondArchiveRequested.Store(true)
		// Even after the first archive has been prepared, every live root stays
		// intact until the complete batch is ready to publish.
		assert.NoFileExists(t, filepath.Join(docsRoot, "new.html"))
		assert.FileExists(t, filepath.Join(docsRoot, "old.html"))
		body := []byte("invalid archive")
		if secondArchiveValid.Load() {
			body = archive.Bytes()
		}
		_, err := w.Write(body)
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

	// A later member's failure must not discard an earlier verified archive.
	// Retry the real batch with the server repaired and prove the first member
	// is consumed from retained staging rather than fetched a second time.
	downloads, err := config.DownloadsDir()
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(downloads, checksum))
	entries, err := os.ReadDir(downloads)
	require.NoError(t, err)
	for _, entry := range entries {
		assert.False(t, entry.IsDir(), "private preparation scratch must be removed after failure")
	}
	secondArchiveValid.Store(true)
	require.NoError(t, InstallComponents(t.Context(), toolchainName, []string{"docs", "stdx-docs"}, true, quietLifecycleOptions()))
	assert.EqualValues(t, 1, docsRequests.Load())
	assert.FileExists(t, filepath.Join(docsRoot, "new.html"))
	assert.FileExists(t, filepath.Join(roots.DocsDir, "stdx", "new.html"))
	entries, err = os.ReadDir(downloads)
	require.NoError(t, err)
	assert.Empty(t, entries, "a successful batch releases its archives and scratch")
}

type componentInstallSink func(progress.Event)

func (f componentInstallSink) Report(e progress.Event) { f(e) }

func TestComponentPreparationAllowsRecoveryAndRejectsConcurrentChanges(t *testing.T) {
	for _, change := range []bool{false, true} {
		t.Run(fmt.Sprint(change), func(t *testing.T) {
			tcDir := installedToolchainHome(t, "lts-1.0.5")
			componentServer(t, map[string]string{"index.html": "docs"}, nil, nil)
			observed := false
			opts := Options{Progress: componentInstallSink(func(e progress.Event) {
				if e.Kind != progress.FetchingComponent {
					return
				}
				observed = true
				ctx, cancel := context.WithTimeout(t.Context(), time.Second)
				defer cancel()
				require.NoError(t, toolchain.RecoverHomeContext(ctx), "proxy recovery must proceed during downloads")
				if change {
					lock, err := toolchain.LockHome(t.Context())
					require.NoError(t, err)
					require.NoError(t, component.WriteManifest(tcDir, component.Stdx, []string{"user-owned"}))
					require.NoError(t, lock.Close())
				}
			})}
			err := InstallComponents(t.Context(), "lts-1.0.5", []string{"docs"}, false, opts)
			assert.True(t, observed)
			if change {
				require.ErrorContains(t, err, "changed during component installation")
				assert.False(t, component.IsInstalled(tcDir, component.Docs))
				assert.True(t, component.IsInstalled(tcDir, component.Stdx))
			} else {
				require.NoError(t, err)
				assert.True(t, component.IsInstalled(tcDir, component.Docs))
			}
		})
	}
}
