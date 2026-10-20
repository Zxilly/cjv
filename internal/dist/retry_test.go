package dist

import (
	"context"
	"crypto/sha256"
	"errors"
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
	"github.com/stretchr/testify/assert"
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

func TestPermanentClientErrors_PreventRetry(t *testing.T) {
	// These errors mean the request itself is wrong; retrying with the
	// same URL/credentials will always fail.
	permanentCodes := []int{400, 401, 403, 404, 405, 410}
	for _, code := range permanentCodes {
		assert.True(t, isNonRetriableHTTPStatus(code),
			"HTTP %d should not be retried (permanent client error)", code)
	}
}

func TestTransientClientErrors_AllowRetry(t *testing.T) {
	// 429 Too Many Requests: temporary rate limiting, will succeed after backoff.
	// 408 Request Timeout: connection issue, may succeed on retry.
	assert.False(t, isNonRetriableHTTPStatus(429),
		"HTTP 429 should be retried (rate limit is temporary)")
	assert.False(t, isNonRetriableHTTPStatus(408),
		"HTTP 408 should be retried (timeout may be transient)")
}

func TestServerErrors_AllowRetry(t *testing.T) {
	// Server errors (5xx) are often transient (deployment, overload, etc.).
	serverCodes := []int{500, 502, 503, 504}
	for _, code := range serverCodes {
		assert.False(t, isNonRetriableHTTPStatus(code),
			"HTTP %d should be retried (server may recover)", code)
	}
}

func TestSuccessAndRedirectCodes_AreNotFlagged(t *testing.T) {
	// Non-error codes should never be flagged as non-retriable.
	assert.False(t, isNonRetriableHTTPStatus(200))
	assert.False(t, isNonRetriableHTTPStatus(301))
	assert.False(t, isNonRetriableHTTPStatus(304))
}

func TestNonRetriableError_ErrorAndUnwrap(t *testing.T) {
	inner := errors.New("resource not found")
	nre := &nonRetriableError{err: inner}

	assert.Equal(t, "resource not found", nre.Error())
	assert.Equal(t, inner, nre.Unwrap())
	assert.True(t, errors.Is(nre, inner))
}

func TestGetMaxDownloadRetriesFromEnv(t *testing.T) {
	t.Setenv("CJV_MAX_RETRIES", "0")
	assert.Equal(t, 0, getMaxDownloadRetries())

	t.Setenv("CJV_MAX_RETRIES", "invalid")
	assert.Equal(t, 3, getMaxDownloadRetries())
}
