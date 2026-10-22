package dist

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/require"
)

func TestMetadataRetriesTransientFailureAndStopsOnNotFound(t *testing.T) {
	t.Setenv(config.EnvMaxRetries, "2")
	for _, status := range []int{503, 404} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			var requests atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				if requests.Add(1) == 1 {
					w.WriteHeader(status)
					return
				}
				_, _ = w.Write([]byte("valid metadata"))
			}))
			defer server.Close()
			data, err := FetchMetadata(t.Context(), server.URL)
			if status == 404 {
				require.Error(t, err)
				require.EqualValues(t, 1, requests.Load())
				var fetchErr *ManifestFetchError
				require.ErrorAs(t, err, &fetchErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, "valid metadata", string(data))
				require.EqualValues(t, 2, requests.Load())
			}
		})
	}
}

func TestReleaseRedirectRetriesAndHonorsOffline(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodHead {
			t.Errorf("expected HEAD, got %s", r.Method)
		}
		if requests.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Location", "/releases/tag/v1.0.0")
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()
	location, err := FetchRedirectLocation(t.Context(), server.URL)
	require.NoError(t, err)
	require.Equal(t, "/releases/tag/v1.0.0", location)
	_, err = FetchRedirectLocation(WithOffline(t.Context(), true), server.URL)
	require.ErrorContains(t, err, "offline mode")
	require.EqualValues(t, 2, requests.Load())
}

func TestMalformedManifestPreservesUsableOfflineCopy(t *testing.T) {
	for _, broken := range []string{"broken manifest", sourceTestManifest("%"), sourceTestManifest("file:///sdk.zip")} {
		t.Run(broken[:8], func(t *testing.T) {
			testInvalidManifestKeepsCache(t, broken)
		})
	}
}

func testInvalidManifestKeepsCache(t *testing.T, broken string) {
	t.Helper()
	configtest.Isolate(t, t.TempDir())
	var malformed atomic.Bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if malformed.Load() {
			_, _ = w.Write([]byte(broken))
			return
		}
		_, _ = w.Write([]byte(sourceTestManifest("sdk.tar.gz")))
	}))
	defer server.Close()
	ctx := withMetadataCache(t.Context())
	_, err := FetchManifest(ctx, server.URL)
	require.NoError(t, err)
	malformed.Store(true)
	_, err = FetchManifest(ctx, server.URL)
	require.Error(t, err)
	manifest, err := FetchManifest(WithOffline(ctx, true), server.URL)
	require.NoError(t, err)
	require.Equal(t, "1.0.0", manifest.Channels.LTS.Latest)
}

func TestLegacyNightlyAndChecksumRemainAvailableOffline(t *testing.T) {
	configtest.Isolate(t, t.TempDir())
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		switch r.URL.Path {
		case "/versions.json":
			_, _ = w.Write([]byte(sourceTestManifestWithNightly("")))
		case "/nightly/nightly.tar.gz.sha256":
			_, _ = w.Write([]byte(sourceTestSHA256))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()
	settings := config.DefaultSettings()
	settings.DistServer = server.URL
	source, err := NewSource(&settings)
	require.NoError(t, err)
	_, err = source.ResolveToolchain(t.Context(), toolchain.Nightly, "", "linux-x64")
	require.NoError(t, err)
	onlineRequests := requests.Load()
	source, err = NewSource(&settings)
	require.NoError(t, err)
	release, err := source.ResolveToolchain(WithOffline(t.Context(), true), toolchain.Nightly, "", "linux-x64")
	require.NoError(t, err)
	require.Equal(t, sourceTestSHA256, release.Download.SHA256)
	require.Equal(t, onlineRequests, requests.Load())
}

func TestReadOnlyMetadataAndCachePrune(t *testing.T) {
	configtest.Isolate(t, t.TempDir())
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("metadata")) }))
	defer server.Close()
	directory, err := CacheDirectory()
	require.NoError(t, err)
	ctx := withMetadataCache(t.Context())
	_, err = FetchMetadata(WithCacheReadOnly(ctx), server.URL)
	require.NoError(t, err)
	require.NoDirExists(t, directory)
	_, err = FetchMetadata(ctx, server.URL)
	require.NoError(t, err)
	entries, err := ListCache(directory)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	removed, err := PruneCache(t.Context(), directory, time.Now().Add(-time.Hour))
	require.NoError(t, err)
	require.Zero(t, removed)
	removed, err = PruneCache(t.Context(), directory, time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.Equal(t, 1, removed)
	entries, err = ListCache(directory)
	require.NoError(t, err)
	require.Empty(t, entries)
	require.FileExists(t, directory+".lock")
}

func TestOfflineMetadataUsesSourceSpecificCache(t *testing.T) {
	configtest.Isolate(t, t.TempDir())
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = w.Write([]byte("metadata"))
	}))
	defer server.Close()
	ctx := withMetadataCache(t.Context())
	_, err := FetchMetadata(ctx, server.URL+"/a")
	require.NoError(t, err)
	data, err := FetchMetadata(WithOffline(ctx, true), server.URL+"/a")
	require.NoError(t, err)
	require.Equal(t, "metadata", string(data))
	_, err = FetchMetadata(WithOffline(ctx, true), server.URL+"/b")
	require.ErrorContains(t, err, "not cached")
	_, err = FetchMetadata(WithOffline(t.Context(), true), server.URL)
	require.ErrorContains(t, err, "offline mode")
	require.EqualValues(t, 1, requests.Load())
}

func TestPersistentArtifactSurvivesSuccessAndRejectsCorruptionOffline(t *testing.T) {
	home := t.TempDir()
	payload := []byte("verified archive")
	digest := strings.ToUpper(fmt.Sprintf("%x", sha256.Sum256(payload)))
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { requests.Add(1); _, _ = w.Write(payload) }))
	defer server.Close()
	downloads := filepath.Join(home, "downloads")
	p, err := BeginPreparation(t.Context(), downloads)
	require.NoError(t, err)
	archive, err := p.Download(t.Context(), server.URL, digest, "sdk.zip", nil)
	require.NoError(t, err)
	p.Complete()
	require.NoError(t, p.Close())
	require.NoFileExists(t, archive)
	p, err = BeginPreparation(t.Context(), downloads)
	require.NoError(t, err)
	_, err = p.Download(WithOffline(t.Context(), true), server.URL, digest, "sdk.zip", nil)
	require.NoError(t, err)
	p.Complete()
	require.NoError(t, p.Close())
	cachePath := filepath.Join(home, "cache", cacheID("artifact", server.URL, digest)+".data")
	require.NoError(t, os.WriteFile(cachePath, []byte("corrupted"), 0o600))
	p, err = BeginPreparation(t.Context(), downloads)
	require.NoError(t, err)
	defer p.Close()
	_, err = p.Download(WithOffline(t.Context(), true), server.URL, digest, "sdk.zip", nil)
	require.Error(t, err)
	require.EqualValues(t, 1, requests.Load())
}
