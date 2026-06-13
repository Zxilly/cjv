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

func TestNightlyReleaseDownloadURLTarget(t *testing.T) {
	url, err := (NightlyRelease{
		TagName: "1.1.0-alpha.20260429010057",
		Version: "1.1.0-alpha.20260429010057",
	}).DownloadURL("https://example.com/releases/download", "win32-x64-ohos-arm32")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/releases/download/1.1.0-alpha.20260429010057/cangjie-sdk-windows-x64-ohos-arm32-1.1.0-alpha.20260429010057.zip", url)
}

func TestNightlyReleaseDownloadURLUsesTagAndAssetVersion(t *testing.T) {
	url, err := (NightlyRelease{
		TagName: "1.1.0-alpha.20260613020028",
		Version: "1.2.0-alpha.20260613020028",
	}).DownloadURL("https://example.com/releases/download", "win32-x64-ohos-arm32")
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/releases/download/1.1.0-alpha.20260613020028/cangjie-sdk-windows-x64-ohos-arm32-1.2.0-alpha.20260613020028.zip", url)
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

	got, err := FetchNightlySHA256(context.Background(), server.URL+"/sdk.zip")
	require.NoError(t, err)
	assert.Equal(t, digest, got)
}

func TestFetchNightlySHA256MissingAndMalformed(t *testing.T) {
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

	// A genuine 404 means no checksum was published: empty digest, no error.
	got, err := FetchNightlySHA256(context.Background(), server.URL+"/missing.zip")
	require.NoError(t, err)
	assert.Empty(t, got)

	// A malformed sidecar must surface an error rather than silently disabling
	// integrity verification.
	_, err = FetchNightlySHA256(context.Background(), server.URL+"/invalid.zip")
	assert.Error(t, err)

	// A network failure must also surface an error, not an empty digest.
	server.Close()
	_, err = FetchNightlySHA256(context.Background(), server.URL+"/network-failure.zip")
	assert.Error(t, err)
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

func TestFetchLatestNightlyReleaseUsesSDKAssetVersion(t *testing.T) {
	const tag = "1.1.0-alpha.20260613020028"
	const assetVersion = "1.2.0-alpha.20260613020028"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"tag_name":"` + tag + `",
			"assets":[
				{"name":"cangjie-docs-html-` + assetVersion + `.tar.gz"},
				{"name":"cangjie-sdk-linux-x64-` + assetVersion + `-sanitizer.tar.gz"},
				{"name":"cangjie-sdk-windows-x64-ohos-` + assetVersion + `.zip.sha256"}
			]
		}`))
	}))
	defer server.Close()

	release, err := FetchLatestNightlyRelease(context.Background(), server.URL, "test-token")
	require.NoError(t, err)
	assert.Equal(t, tag, release.TagName)
	assert.Equal(t, assetVersion, release.Version)

	latest, err := FetchLatestNightly(context.Background(), server.URL, "test-token")
	require.NoError(t, err)
	assert.Equal(t, assetVersion, latest)
}

func TestFetchLatestNightlyReleaseRejectsMixedSDKAssetVersions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{
			"tag_name":"1.1.0-alpha.20260613020028",
			"assets":[
				{"name":"cangjie-sdk-linux-x64-1.2.0-alpha.20260613020028.tar.gz"},
				{"name":"cangjie-sdk-windows-x64-1.3.0-alpha.20260613020028.zip"}
			]
		}`))
	}))
	defer server.Close()

	_, err := FetchLatestNightlyRelease(context.Background(), server.URL, "test-token")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "multiple SDK asset versions")
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
