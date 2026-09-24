//go:build !mirror

package selfupdate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"testing"

	"github.com/Zxilly/cjv/internal/dist"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateReturnsDetectLatestError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := Update(ctx, "https://github.com/Zxilly/cjv/releases", "1.0.0", nil)

	require.Error(t, err)
}

func updateTestBinaryName() string { return "cjv" }

type releaseTransport func(*http.Request) (*http.Response, error)

func (f releaseTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func serveUpdateRelease(t *testing.T, tag string, archive []byte, digest string) string {
	t.Helper()
	assetName := releaseAssetName(updateTestBinaryName(), runtime.GOOS, runtime.GOARCH)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/repos/owner/repo/releases/latest":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tag_name": tag,
				"assets": []map[string]string{
					{"name": assetName, "browser_download_url": "http://" + r.Host + "/" + assetName},
					{"name": "checksums.txt", "browser_download_url": "http://" + r.Host + "/checksums.txt"},
				},
			})
		case "/" + assetName:
			_, _ = w.Write(archive)
		case "/checksums.txt":
			_, _ = fmt.Fprintf(w, "%s  %s\n", digest, assetName)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	target, err := url.Parse(server.URL)
	require.NoError(t, err)
	client := dist.HTTPClient()
	previousTransport := client.Transport
	client.Transport = releaseTransport(func(req *http.Request) (*http.Response, error) {
		clone := req.Clone(req.Context())
		if clone.URL.Host == "api.github.com" {
			clone.URL.Scheme, clone.URL.Host = target.Scheme, target.Host
		}
		if clone.URL.Host != target.Host {
			return nil, fmt.Errorf("unexpected non-local update URL: %s", clone.URL)
		}
		return http.DefaultTransport.RoundTrip(clone)
	})
	t.Cleanup(func() { client.Transport = previousTransport })
	return "https://github.com/owner/repo/releases"
}

func TestExtractSlugFallbacks(t *testing.T) {
	assert.Equal(t, "owner/repo", extractSlug("https://github.com/owner/repo/releases"))
	assert.Equal(t, "https://github.com/owner-only", extractSlug("https://github.com/owner-only"))
	assert.Equal(t, "not a url", extractSlug("not a url"))
}
