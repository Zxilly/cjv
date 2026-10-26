package dist

import (
	"context"
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/stretchr/testify/require"
)

func TestMetadataDoesNotWaitForArtifactDownload(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	payload := []byte("verified SDK")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/artifact" {
			close(started)
			<-release
			_, _ = w.Write(payload)
			return
		}
		_, _ = w.Write([]byte("metadata"))
	}))
	t.Cleanup(func() { unblock(); server.Close() })
	done := make(chan error, 1)
	go func() {
		_, err := downloadPersistent(t.Context(), filepath.Join(home, "cache"), filepath.Join(home, "downloads"), server.URL+"/artifact", fmt.Sprintf("%x", sha256.Sum256(payload)), "sdk", nil)
		done <- err
	}()
	<-started
	// The server keeps the transfer active until the independent metadata
	// operation completes. The deadline is only a deadlock guard.
	ctx, cancel := context.WithTimeout(withMetadataCache(t.Context()), 5*time.Second)
	defer cancel()
	data, err := FetchMetadata(ctx, server.URL+"/metadata")
	unblock()
	require.NoError(t, <-done)
	require.NoError(t, err)
	require.Equal(t, "metadata", string(data))
}

func TestConcurrentCacheRequestsShareOneArtifactTransfer(t *testing.T) {
	home := t.TempDir()
	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	var requests atomic.Int32
	payload := []byte("one verified transfer")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) == 1 {
			close(started)
		}
		<-release
		_, _ = w.Write(payload)
	}))
	t.Cleanup(func() { unblock(); server.Close() })
	digest := fmt.Sprintf("%x", sha256.Sum256(payload))
	done := make(chan error, 2)
	download := func(staging string) {
		_, err := downloadPersistent(t.Context(), filepath.Join(home, "cache"), filepath.Join(home, staging), server.URL, digest, "sdk", nil)
		done <- err
	}
	go download("first")
	<-started
	go download("second")
	unblock()
	require.NoError(t, <-done)
	require.NoError(t, <-done)
	require.EqualValues(t, 1, requests.Load())
	for _, staging := range []string{"first", "second"} {
		data, err := os.ReadFile(filepath.Join(home, staging, digest))
		require.NoError(t, err)
		require.Equal(t, payload, data)
	}
}

func TestCacheCleanWaitsForActiveEntryWithoutHoldingGlobalLock(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	directory := filepath.Join(home, "cache")
	require.NoError(t, os.MkdirAll(directory, 0o700))
	started, release := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	payload := []byte("verified replacement")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		close(started)
		<-release
		_, _ = w.Write(payload)
	}))
	t.Cleanup(func() { unblock(); server.Close() })
	digest := fmt.Sprintf("%x", sha256.Sum256(payload))
	path := filepath.Join(directory, cacheID("artifact", server.URL, digest)+".data")
	require.NoError(t, os.WriteFile(path, []byte("old incomplete data"), 0o600))
	done := make(chan error, 1)
	go func() {
		_, err := downloadPersistent(t.Context(), directory, filepath.Join(home, "downloads"), server.URL, digest, "sdk", nil)
		done <- err
	}()
	<-started
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	removed, err := PruneCache(ctx, directory, time.Time{})
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Zero(t, removed)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, "old incomplete data", string(data), "clean must not touch a writer's active entry")
	// Its canceled wait must also leave cleanup coordination available.
	global, err := lockCache(t.Context(), directory)
	require.NoError(t, err)
	require.NoError(t, global.Close())
	unblock()
	require.NoError(t, <-done)
	removed, err = PruneCache(t.Context(), directory, time.Time{})
	require.NoError(t, err)
	require.Equal(t, 1, removed)
	require.NoFileExists(t, path)
	require.FileExists(t, filepath.Join(home, "downloads", digest), "clean must preserve the installation's private archive")
}

func TestSuccessfulMetadataReadSurvivesCachePublicationFailure(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	require.NoError(t, os.WriteFile(filepath.Join(home, "cache"), []byte("not a directory"), 0o600))
	data, err := cachedMetadata(t.Context(), "https://example.invalid/versions.json", func() ([]byte, error) {
		return []byte("validated metadata"), nil
	})
	require.NoError(t, err)
	require.Equal(t, "validated metadata", string(data))
}
