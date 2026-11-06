package dist

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/progress"
	"github.com/stretchr/testify/require"
)

func TestDownloadRejectsUnsolicitedPartialResponse(t *testing.T) {
	t.Setenv("CJV_MAX_RETRIES", "0")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Range", "bytes 5-9/10")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte("56789"))
	}))
	defer server.Close()
	dest := filepath.Join(t.TempDir(), "archive")
	err := DownloadFile(t.Context(), server.URL, dest, "", nil)
	require.Error(t, err)
	require.NoFileExists(t, dest, "a suffix of an artifact must never be published as the whole file")
}

func TestDownloadValidatesResumedRange(t *testing.T) {
	for _, header := range []string{"bytes 5-invalid/10", "bytes 5-4/10", "bytes 5-9/9", "bytes 5-9/invalid", "bytes 5-9/10", "bytes 5-5/10"} {
		t.Run(header, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Range", header)
				w.WriteHeader(http.StatusPartialContent)
				_, _ = w.Write([]byte("56"))
			}))
			defer server.Close()
			partial := filepath.Join(t.TempDir(), "partial")
			require.NoError(t, os.WriteFile(partial, []byte("01234"), 0o644))
			err := downloadOnce(t.Context(), server.URL, partial, "", "", progress.Discard)
			require.Error(t, err, "invalid or incomplete ranges must not report success")
		})
	}
}

func TestDownloadContinuesLimitedRangeResponse(t *testing.T) {
	partial := filepath.Join(t.TempDir(), "partial")
	require.NoError(t, os.WriteFile(partial, []byte("012"), 0o644))
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		start, end := 3, 5
		if requests == 2 {
			start, end = 6, 9
		}
		if r.Header.Get("Range") != fmt.Sprintf("bytes=%d-", start) {
			t.Errorf("unexpected range: %s", r.Header.Get("Range"))
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/10", start, end))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte("0123456789")[start : end+1])
	}))
	defer server.Close()
	err := retryTransfer(t.Context(), func() error {
		return downloadOnce(t.Context(), server.URL, partial, "", "", progress.Discard)
	})
	require.NoError(t, err)
	data, err := os.ReadFile(partial)
	require.NoError(t, err)
	require.Equal(t, "0123456789", string(data))
	require.Equal(t, 2, requests)
}
