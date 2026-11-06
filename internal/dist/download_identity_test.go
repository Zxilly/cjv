package dist

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUnverifiedDownloadRetryDoesNotCombineDifferentArtifacts(t *testing.T) {
	archive := func(value string) []byte {
		var data bytes.Buffer
		w := zip.NewWriter(&data)
		f, err := w.CreateHeader(&zip.FileHeader{Name: "sdk.txt", Method: zip.Store})
		require.NoError(t, err)
		_, err = f.Write([]byte(strings.Repeat(value, 100)))
		require.NoError(t, err)
		require.NoError(t, w.Close())
		return data.Bytes()
	}
	old, current := archive("old"), archive("new")
	require.Equal(t, len(old), len(current))
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requests.Add(1) == 1 {
			w.Header().Set("Content-Length", strconv.Itoa(len(old)))
			_, _ = w.Write(old[:len(old)/2])
			return // Truncate the first response, then republish the URL.
		}
		if value := r.Header.Get("Range"); value != "" {
			var start int
			_, err := fmt.Sscanf(value, "bytes=%d-", &start)
			if err != nil || start < 0 || start >= len(current) {
				http.Error(w, "invalid range", http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, len(current)-1, len(current)))
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write(current[start:])
			return
		}
		_, _ = w.Write(current)
	}))
	defer server.Close()
	t.Setenv("CJV_MAX_RETRIES", "1")
	path, err := DownloadCached(t.Context(), server.URL+"/nightly.zip", "", t.TempDir(), nil)
	require.NoError(t, err)
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	require.Equal(t, current, data, "without an entity validator, a retry must fetch one complete response")
	require.Equal(t, int32(2), requests.Load())
}
