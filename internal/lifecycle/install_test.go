package lifecycle

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func quietLifecycleOptions() Options {
	return Options{}
}

func TestResolveTargetToolchainUsesManifestNightlyVersion(t *testing.T) {
	const version = "1.2.0-alpha.20260613020028"
	const sha = "0000000000000000000000000000000000000000000000000000000000000000"

	manifest := dist.Manifest{}
	manifest.Channels.LTS = dist.ChannelInfo{Latest: "1.0.5", Versions: map[string]map[string]dist.DownloadInfo{
		"1.0.5": {"linux-x64": {Name: "lts.tar.gz", URL: "https://example/lts.tar.gz", SHA256: sha}},
	}}
	manifest.Channels.STS = dist.ChannelInfo{Latest: "1.1.0", Versions: map[string]map[string]dist.DownloadInfo{
		"1.1.0": {"linux-x64": {Name: "sts.tar.gz", URL: "https://example/sts.tar.gz", SHA256: sha}},
	}}
	nightly := dist.ChannelInfo{Latest: version, Versions: map[string]map[string]dist.DownloadInfo{
		version: {
			"linux-x64":      {Name: "host.tar.gz", URL: "https://example/host.tar.gz", SHA256: sha},
			"linux-x64-ohos": {Name: "ohos.tar.gz", URL: "https://example/ohos.tar.gz", SHA256: sha},
		},
	}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/nightly.json" {
			require.NoError(t, json.NewEncoder(w).Encode(nightly))
			return
		}
		require.NoError(t, json.NewEncoder(w).Encode(manifest))
	}))
	defer server.Close()

	settings := config.DefaultSettings()
	settings.DefaultHost = "linux-amd64"
	settings.ManifestURL = server.URL
	fetcher, err := NewManifestFetcherForSettings(&settings, quietLifecycleOptions())
	require.NoError(t, err)

	host, err := ResolveAndLocate(context.Background(), toolchain.ToolchainName{Channel: toolchain.Nightly}, &settings, fetcher)
	require.NoError(t, err)
	assert.Equal(t, "nightly-"+version, host.Name)

	target, err := resolveTargetToolchain(
		context.Background(),
		toolchain.ToolchainName{Channel: toolchain.Nightly, Version: version},
		&settings,
		fetcher,
		"ohos",
	)
	require.NoError(t, err)
	assert.Equal(t, "nightly-"+version+"-linux-x64-ohos", target.Name)
	assert.True(t, strings.HasSuffix(target.URL, "/ohos.tar.gz"), target.URL)
}
