package dist

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNightlyDownloadURL(t *testing.T) {
	url, err := NightlyDownloadURL("https://example.com/releases/download", "1.1.0-alpha.20260306010001", "windows", "amd64")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/releases/download/1.1.0-alpha.20260306010001/cangjie-sdk-windows-x64-1.1.0-alpha.20260306010001.zip", url)
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


