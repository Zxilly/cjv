package dist

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDownloadFile(t *testing.T) {
	content := []byte("fake sdk archive content")
	hash := fmt.Sprintf("%x", sha256.Sum256(content))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(content)
	}))
	defer server.Close()

	tmp := t.TempDir()
	dest := filepath.Join(tmp, "archive.zip")

	err := DownloadFile(context.Background(), server.URL+"/test.zip", dest, hash)
	require.NoError(t, err)

	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestDownloadFileBadChecksum(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("content"))
	}))
	defer server.Close()

	tmp := t.TempDir()
	dest := filepath.Join(tmp, "archive.zip")

	err := DownloadFile(context.Background(), server.URL+"/test.zip", dest, "0000000000000000000000000000000000000000000000000000000000000000")
	assert.Error(t, err)
}

func TestDownloadFileSkipChecksum(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("nightly content"))
	}))
	defer server.Close()

	tmp := t.TempDir()
	dest := filepath.Join(tmp, "archive.tar.gz")

	// Empty sha256 skips verification (nightly scenario)
	err := DownloadFile(context.Background(), server.URL+"/test.tar.gz", dest, "")
	require.NoError(t, err)
}

func TestDownloadFilePreservesExistingDestinationOnChecksumFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("bad content"))
	}))
	defer server.Close()

	tmp := t.TempDir()
	dest := filepath.Join(tmp, "archive.zip")
	require.NoError(t, os.WriteFile(dest, []byte("verified cache"), 0o644))

	err := DownloadFile(context.Background(), server.URL+"/test.zip", dest, "0000000000000000000000000000000000000000000000000000000000000000")
	require.Error(t, err)

	data, readErr := os.ReadFile(dest)
	require.NoError(t, readErr)
	assert.Equal(t, []byte("verified cache"), data)
}

func TestDownloadFileRetriesTooManyRequests(t *testing.T) {
	content := []byte("recovered content")
	hash := fmt.Sprintf("%x", sha256.Sum256(content))
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte("slow down"))
			return
		}
		_, _ = w.Write(content)
	}))
	defer server.Close()

	tmp := t.TempDir()
	dest := filepath.Join(tmp, "archive.zip")

	require.NoError(t, DownloadFile(context.Background(), server.URL+"/test.zip", dest, hash))
	assert.GreaterOrEqual(t, attempts.Load(), int32(2))
}

func TestDownloadFileCached_CacheHit(t *testing.T) {
	cacheDir := t.TempDir()
	content := []byte("cached content")
	h := sha256.Sum256(content)
	sha256Hex := hex.EncodeToString(h[:])
	cachedPath := filepath.Join(cacheDir, sha256Hex)
	require.NoError(t, os.WriteFile(cachedPath, content, 0o644))

	dest := filepath.Join(t.TempDir(), "output.tar.gz")
	err := DownloadFileCached(context.Background(), "https://example.com/nonexistent", dest, sha256Hex, cacheDir)
	require.NoError(t, err)

	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

func TestDownloadFileCached_CacheMiss(t *testing.T) {
	content := []byte("fresh download content")
	h := sha256.Sum256(content)
	sha256Hex := hex.EncodeToString(h[:])

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(content)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	dest := filepath.Join(t.TempDir(), "output.tar.gz")

	err := DownloadFileCached(context.Background(), server.URL+"/test.tar.gz", dest, sha256Hex, cacheDir)
	require.NoError(t, err)

	// dest should have the content.
	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, content, data)

	// Cache file should also exist.
	cachedData, err := os.ReadFile(filepath.Join(cacheDir, sha256Hex))
	require.NoError(t, err)
	assert.Equal(t, content, cachedData)
}

func TestDownloadFileCached_CacheHitChecksumMismatchRedownloads(t *testing.T) {
	content := []byte("fresh download content")
	h := sha256.Sum256(content)
	sha256Hex := hex.EncodeToString(h[:])

	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		_, _ = w.Write(content)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	cachedPath := filepath.Join(cacheDir, sha256Hex)
	require.NoError(t, os.WriteFile(cachedPath, []byte("corrupt cache"), 0o644))

	dest := filepath.Join(t.TempDir(), "output.tar.gz")
	err := DownloadFileCached(context.Background(), server.URL+"/test.tar.gz", dest, sha256Hex, cacheDir)
	require.NoError(t, err)

	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, content, data)

	cachedData, err := os.ReadFile(cachedPath)
	require.NoError(t, err)
	assert.Equal(t, content, cachedData)
	assert.Equal(t, int32(1), requests.Load(), "checksum mismatch should trigger a redownload")
}

func TestDownloadFileCached_NoChecksumUsesURLHash(t *testing.T) {
	content := []byte("nightly content no checksum")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(content)
	}))
	defer server.Close()

	cacheDir := t.TempDir()
	dest := filepath.Join(t.TempDir(), "output.tar.gz")
	url := server.URL + "/nightly.tar.gz"

	err := DownloadFileCached(context.Background(), url, dest, "", cacheDir)
	require.NoError(t, err)

	// Cache key should be sha256(url).
	urlHash := sha256.Sum256([]byte(url))
	expectedKey := hex.EncodeToString(urlHash[:])
	_, err = os.Stat(filepath.Join(cacheDir, expectedKey))
	assert.NoError(t, err)
}

func TestDownloadOnce_ResumeWithRange(t *testing.T) {
	// Full content split into two halves.
	full := []byte("AAAAABBBBB")
	first := full[:5]
	second := full[5:]

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")
		if rangeHeader != "" {
			// Parse "bytes=N-"
			var start int64
			fmt.Sscanf(rangeHeader, "bytes=%d-", &start)
			if start >= int64(len(full)) {
				w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
				return
			}
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, len(full)-1, len(full)))
			w.Header().Set("Content-Length", strconv.Itoa(len(full)-int(start)))
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write(full[start:])
			return
		}
		w.Header().Set("Content-Length", strconv.Itoa(len(full)))
		_, _ = w.Write(full)
	}))
	defer server.Close()

	hash := fmt.Sprintf("%x", sha256.Sum256(full))

	// Create a partial file with the first half.
	tmpDir := t.TempDir()
	partialPath := filepath.Join(tmpDir, "test.partial")
	require.NoError(t, os.WriteFile(partialPath, first, 0o644))

	// downloadOnce should resume and append the second half.
	err := downloadOnce(context.Background(), server.URL+"/test.zip", partialPath, "test.zip", hash)
	require.NoError(t, err)

	data, err := os.ReadFile(partialPath)
	require.NoError(t, err)
	assert.Equal(t, full, data)
	_ = second // referenced for clarity
}

func TestDownloadOnce_ResumeServerReturns200(t *testing.T) {
	// Server does not support Range — returns full content with 200.
	full := []byte("complete content here")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(full)
	}))
	defer server.Close()

	hash := fmt.Sprintf("%x", sha256.Sum256(full))

	tmpDir := t.TempDir()
	partialPath := filepath.Join(tmpDir, "test.partial")
	// Write some garbage as partial.
	require.NoError(t, os.WriteFile(partialPath, []byte("old"), 0o644))

	err := downloadOnce(context.Background(), server.URL+"/test.zip", partialPath, "test.zip", hash)
	require.NoError(t, err)

	data, err := os.ReadFile(partialPath)
	require.NoError(t, err)
	assert.Equal(t, full, data)
}

// --- Tests merged from download_replace_test.go ---

func TestDownloadFileReplacesExistingDestination(t *testing.T) {
	content := []byte("fresh nightly archive")
	hash := fmt.Sprintf("%x", sha256.Sum256(content))

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(content)
	}))
	defer server.Close()

	tmp := t.TempDir()
	dest := filepath.Join(tmp, "archive.zip")
	require.NoError(t, os.WriteFile(dest, []byte("stale content"), 0o644))

	require.NoError(t, DownloadFile(context.Background(), server.URL+"/test.zip", dest, hash))

	data, err := os.ReadFile(dest)
	require.NoError(t, err)
	assert.Equal(t, content, data)
}

// --- Tests merged from http_retry_test.go ---

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

// --- Tests merged from quick_coverage_test.go (nonRetriableError is in download.go) ---

func TestNonRetriableError_ErrorAndUnwrap(t *testing.T) {
	inner := errors.New("resource not found")
	nre := &nonRetriableError{err: inner}

	assert.Equal(t, "resource not found", nre.Error())
	assert.Equal(t, inner, nre.Unwrap())
	assert.True(t, errors.Is(nre, inner))
}
