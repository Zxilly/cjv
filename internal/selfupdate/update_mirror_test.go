//go:build mirror

package selfupdate

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateGitCodeReturnsFetchError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := Update(ctx, "https://gitcode.com/Zxilly/cjv/releases", "1.0.0")

	require.Error(t, err)
}

func TestMirrorAssetName(t *testing.T) {
	assert.Equal(t, "cjv-mirror_linux_amd64.tar.gz", mirrorAssetName("linux", "amd64"))
	assert.Equal(t, "cjv-mirror_darwin_arm64.tar.gz", mirrorAssetName("darwin", "arm64"))
	assert.Equal(t, "cjv-mirror_windows_amd64.zip", mirrorAssetName("windows", "amd64"))
}

func TestGitCodeReleasesBase(t *testing.T) {
	base, err := gitCodeReleasesBase("https://gitcode.com/Zxilly/cjv/releases")
	require.NoError(t, err)
	assert.Equal(t, "https://gitcode.com/Zxilly/cjv/releases", base)

	base, err = gitCodeReleasesBase("https://gitcode.com/Zxilly/cjv")
	require.NoError(t, err)
	assert.Equal(t, "https://gitcode.com/Zxilly/cjv/releases", base)

	_, err = gitCodeReleasesBase("https://gitcode.com/owner-only")
	require.Error(t, err)
}

func TestTagFromReleaseURL(t *testing.T) {
	assert.Equal(t, "v1.2.3", tagFromReleaseURL("https://gitcode.com/Zxilly/cjv/releases/tag/v1.2.3"))
	assert.Equal(t, "v1.2.3", tagFromReleaseURL("https://gitcode.com/Zxilly/cjv/releases/tag/v1.2.3/"))
	assert.Equal(t, "v0.1.0-rc1", tagFromReleaseURL("/Zxilly/cjv/releases/tag/v0.1.0-rc1?foo=bar"))
	assert.Equal(t, "", tagFromReleaseURL("https://gitcode.com/Zxilly/cjv/releases/latest"))
	assert.Equal(t, "", tagFromReleaseURL("not a url://"))
}

func TestFetchGitCodeLatestTagFollowsRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/releases/latest") {
			http.Redirect(w, r, "/Zxilly/cjv/releases/tag/v9.8.7", http.StatusFound)
			return
		}
		http.NotFound(w, r)
	}))
	defer srv.Close()

	tag, err := fetchGitCodeLatestTag(context.Background(), srv.URL+"/Zxilly/cjv/releases/latest")
	require.NoError(t, err)
	assert.Equal(t, "v9.8.7", tag)
}

func TestFetchGitCodeLatestTagErrorsWithoutRedirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	_, err := fetchGitCodeLatestTag(context.Background(), srv.URL+"/x/cjv/releases/latest")
	require.Error(t, err)
}
