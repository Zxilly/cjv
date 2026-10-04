package dist

import (
	"crypto/sha256"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCachedDownloadResumesAcrossInvocations(t *testing.T) {
	t.Setenv("CJV_MAX_RETRIES", "0")
	content := []byte("0123456789")
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			require.Empty(t, r.Header.Get("Range"))
			w.Header().Set("Content-Length", "10")
			_, _ = w.Write(content[:5])
			return
		}
		require.Equal(t, "bytes=5-", r.Header.Get("Range"))
		w.Header().Set("Content-Range", fmt.Sprintf("bytes 5-9/%d", len(content)))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(content[5:])
	}))
	defer server.Close()
	cache := t.TempDir()
	checksum := fmt.Sprintf("%x", sha256.Sum256(content))
	_, err := DownloadCached(t.Context(), server.URL, checksum, cache, nil)
	require.Error(t, err)
	require.FileExists(t, filepath.Join(cache, checksum+".partial"))
	path, err := DownloadCached(t.Context(), server.URL, checksum, cache, nil)
	require.NoError(t, err)
	actual, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, content, actual)
	require.EqualValues(t, 2, requests.Load())
	require.NoFileExists(t, filepath.Join(cache, checksum+".partial"))
}
