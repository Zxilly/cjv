package dist

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNightlyDownloadURL(t *testing.T) {
	url, err := NightlyDownloadURL("https://example.com/releases/download", "1.1.0-alpha.20260306010001", "windows", "amd64")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/releases/download/1.1.0-alpha.20260306010001/cangjie-sdk-windows-x64-1.1.0-alpha.20260306010001.zip", url)
}

func TestNightlyDownloadURLForPlatformTarget(t *testing.T) {
	url, err := NightlyDownloadURLForPlatform("https://example.com/releases/download", "1.1.0-alpha.20260429010057", "win32-x64-ohos-arm32")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/releases/download/1.1.0-alpha.20260429010057/cangjie-sdk-windows-x64-ohos-arm32-1.1.0-alpha.20260429010057.zip", url)
}

func TestParseSHA256(t *testing.T) {
	digest := strings.Repeat("ab", 32)
	assert.Equal(t, digest, parseSHA256(digest+"\n"))
	assert.Equal(t, digest, parseSHA256(strings.ToUpper(digest)))
	assert.Empty(t, parseSHA256("abc"))
	assert.Empty(t, parseSHA256(digest+" cangjie-sdk.zip"))
	assert.Empty(t, parseSHA256(strings.Repeat("z", 64)))
}

func TestFetchNightlySHA256(t *testing.T) {
	digest := strings.Repeat("ab", 32)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/sdk.zip.sha256", r.URL.Path)
		w.Write([]byte(strings.ToUpper(digest) + "\n"))
	}))
	defer server.Close()

	got := FetchNightlySHA256(context.Background(), server.URL+"/sdk.zip")
	assert.Equal(t, digest, got)
}

func TestFetchNightlySHA256BestEffortFallbacks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/missing.zip.sha256":
			http.NotFound(w, r)
		case "/invalid.zip.sha256":
			w.Write([]byte("not-a-digest"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	assert.Empty(t, FetchNightlySHA256(context.Background(), server.URL+"/missing.zip"))
	assert.Empty(t, FetchNightlySHA256(context.Background(), server.URL+"/invalid.zip"))

	server.Close()
	assert.Empty(t, FetchNightlySHA256(context.Background(), server.URL+"/network-failure.zip"))
}

func TestFetchLatestNightly(t *testing.T) {
	const tag = "1.1.0-alpha.20260306010001"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"tag_name":"` + tag + `"}`))
	}))
	defer server.Close()

	latest, err := FetchLatestNightly(context.Background(), server.URL, "test-token")
	require.NoError(t, err)
	assert.Equal(t, tag, latest)
}

func TestFetchLatestNightlyEmptyTag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":""}`))
	}))
	defer server.Close()

	_, err := FetchLatestNightly(context.Background(), server.URL, "test-token")
	assert.Error(t, err)
}

func TestFetchLatestNightlyInvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	_, err := FetchLatestNightly(context.Background(), server.URL, "test-token")
	assert.Error(t, err)
}
