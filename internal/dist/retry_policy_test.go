package dist

import (
	"context"
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

	"github.com/Zxilly/cjv/internal/progress"
	"github.com/stretchr/testify/require"
)

func TestNightlyChecksumRetryPolicy(t *testing.T) {
	for _, status := range []int{403, 404, 408, 429, 500} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			t.Setenv("CJV_MAX_RETRIES", "3")
			var attempts atomic.Int32
			digest := strings.Repeat("ab", 32)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if attempts.Add(1) == 1 {
					w.WriteHeader(status)
					return
				}
				_, _ = w.Write([]byte(digest))
			}))
			defer server.Close()
			got, err := FetchNightlySHA256(t.Context(), server.URL+"/sdk.zip")
			switch status {
			case 403:
				require.Error(t, err)
				require.EqualValues(t, 1, attempts.Load())
			case 404:
				require.NoError(t, err)
				require.Empty(t, got)
				require.EqualValues(t, 1, attempts.Load())
			default:
				require.NoError(t, err)
				require.Equal(t, digest, got)
				require.EqualValues(t, 2, attempts.Load())
			}
		})
	}
}

func TestNetworkOperationsRespectCanceledContext(t *testing.T) {
	for _, operation := range []string{"sidecar", "download", "cached"} {
		t.Run(operation, func(t *testing.T) {
			t.Setenv("CJV_MAX_RETRIES", "12")
			var attempts atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attempts.Add(1)
				w.WriteHeader(http.StatusInternalServerError)
			}))
			defer server.Close()
			root := t.TempDir()
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			start := time.Now()
			var err error
			switch operation {
			case "sidecar":
				_, err = FetchNightlySHA256(ctx, server.URL+"/sdk.zip")
			case "download":
				err = DownloadFile(ctx, server.URL, filepath.Join(root, "unused", "sdk.zip"), "", nil)
			case "cached":
				_, err = DownloadCached(ctx, server.URL, strings.Repeat("ab", 32), filepath.Join(root, "unused"), nil)
			}
			require.ErrorIs(t, err, context.Canceled)
			require.Less(t, time.Since(start), time.Second, "canceled operations must not wait through backoff")
			require.Zero(t, attempts.Load())
			entries, err := os.ReadDir(root)
			require.NoError(t, err)
			require.Empty(t, entries, "pre-canceled operations must not create download state")
		})
	}
}

func TestCachedDownloadCancellationPreservesResume(t *testing.T) {
	t.Setenv("CJV_MAX_RETRIES", "3")
	content := []byte("0123456789")
	checksum := fmt.Sprintf("%x", sha256.Sum256(content))
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			w.Header().Set("Content-Length", "10")
			_, _ = w.Write(content[:5])
			w.(http.Flusher).Flush()
			<-r.Context().Done()
			return
		}
		if r.Header.Get("Range") != "bytes=5-" {
			t.Errorf("unexpected resume range: %s", r.Header.Get("Range"))
		}
		w.Header().Set("Content-Range", "bytes 5-9/10")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(content[5:])
	}))
	defer server.Close()
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	cache := t.TempDir()
	sink := sinkFunc(func(event progress.Event) {
		if event.Kind == progress.DownloadAdvanced {
			cancel()
		}
	})
	_, err := DownloadCached(ctx, server.URL, checksum, cache, sink)
	require.ErrorIs(t, err, context.Canceled)
	require.EqualValues(t, 1, attempts.Load())
	partial, err := os.ReadFile(filepath.Join(cache, checksum+".partial"))
	require.NoError(t, err)
	require.Equal(t, content[:5], partial)
	path, err := DownloadCached(t.Context(), server.URL, checksum, cache, nil)
	require.NoError(t, err)
	actual, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, content, actual)
	require.EqualValues(t, 2, attempts.Load())
}

func TestNightlyChecksumDeadlineStopsRetries(t *testing.T) {
	t.Setenv("CJV_MAX_RETRIES", "12")
	var attempts atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		<-r.Context().Done()
	}))
	defer server.Close()
	ctx, cancel := context.WithTimeout(t.Context(), 25*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := FetchNightlySHA256(ctx, server.URL+"/sdk.zip")
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.Less(t, time.Since(start), time.Second)
	require.LessOrEqual(t, attempts.Load(), int32(1))
}
