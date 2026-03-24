package dist

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNightlyProbeExists(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	exists, err := ProbeNightlyVersion(context.Background(), server.URL, "1.1.0-alpha.20260306010001", "windows", "amd64")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestNightlyProbeNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	exists, err := ProbeNightlyVersion(context.Background(), server.URL, "99.99.99", "windows", "amd64")
	require.NoError(t, err)
	assert.False(t, exists)
}

func TestNightlyDownloadURL(t *testing.T) {
	url, err := NightlyDownloadURL("https://example.com/releases/download", "1.1.0-alpha.20260306010001", "windows", "amd64")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/releases/download/1.1.0-alpha.20260306010001/cangjie-sdk-windows-x64-1.1.0-alpha.20260306010001.zip", url)
}

func TestFetchLatestNightly(t *testing.T) {
	releases := `[{"tag_name":"1.1.0-alpha.20260306010001"},{"tag_name":"1.1.0-alpha.20260305010001"}]`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(releases))
	}))
	defer server.Close()

	latest, err := FetchLatestNightly(context.Background(), server.URL, "test-token")
	require.NoError(t, err)
	assert.Equal(t, "1.1.0-alpha.20260306010001", latest)
}

func TestFetchLatestNightlyEmpty(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("[]"))
	}))
	defer server.Close()

	_, err := FetchLatestNightly(context.Background(), server.URL, "test-token")
	assert.Error(t, err)
}

func TestFetchLatestNightlyUnsorted(t *testing.T) {
	releases := `[{"tag_name":"1.1.0-alpha.20260301010001"},{"tag_name":"1.1.0-alpha.20260310010001"},{"tag_name":"1.1.0-alpha.20260305010001"}]`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(releases))
	}))
	defer server.Close()

	latest, err := FetchLatestNightly(context.Background(), server.URL, "test-token")
	require.NoError(t, err)
	assert.Equal(t, "1.1.0-alpha.20260310010001", latest)
}

func TestFetchLatestNightlyCrossVersion(t *testing.T) {
	// When the version prefix changes (1.1.0 -> 1.2.0), lexicographic
	// ordering would pick 1.2.0 even if its timestamp is older.
	// Timestamp-based sorting must pick the newest timestamp regardless of version prefix.
	releases := `[
		{"tag_name":"1.2.0-alpha.20260301010001"},
		{"tag_name":"1.1.0-alpha.20260315010001"},
		{"tag_name":"1.1.0-alpha.20260310010001"}
	]`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(releases))
	}))
	defer server.Close()

	latest, err := FetchLatestNightly(context.Background(), server.URL, "test-token")
	require.NoError(t, err)
	assert.Equal(t, "1.1.0-alpha.20260315010001", latest)
}

func TestExtractNightlyTimestamp(t *testing.T) {
	tests := []struct {
		tag  string
		want int64
	}{
		{"1.1.0-alpha.20260306010001", 20260306010001},
		{"1.2.0-beta.20260315120000", 20260315120000},
		{"no-timestamp", 0},
		{"trailing-dot.", 0},
		{"", 0},
	}
	for _, tt := range tests {
		got := extractNightlyTimestamp(tt.tag)
		assert.Equal(t, tt.want, got, "extractNightlyTimestamp(%q)", tt.tag)
	}
}

// --- Tests merged from nightly_probe_test.go ---

func TestNightlyProbeServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer server.Close()

	_, err := ProbeNightlyVersion(context.Background(), server.URL, "1.1.0-alpha.20260306010001", "windows", "amd64")
	require.Error(t, err)
}
