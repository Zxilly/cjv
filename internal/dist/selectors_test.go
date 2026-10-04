package dist

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/require"
)

func TestSourceSelectorsUsePublishedSplitManifests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := func(versions ...string) ChannelInfo {
			channel := ChannelInfo{Latest: versions[0], Versions: map[string]map[string]DownloadInfo{}}
			for _, version := range versions {
				channel.Versions[version] = map[string]DownloadInfo{"linux-x64": {Name: "sdk.zip", URL: "http://" + r.Host + "/sdk.zip", SHA256: strings.Repeat("a", 64)}}
			}
			return channel
		}
		if r.URL.Path == "/nightly.json" {
			_ = json.NewEncoder(w).Encode(info("1.3.0-alpha.20261005000000", "1.3.0-alpha.20261004001050"))
			return
		}
		var manifest Manifest
		manifest.Channels.STS = info("1.2.10", "1.2.1")
		manifest.Channels.LTS = info("1.0.0")
		_ = json.NewEncoder(w).Encode(manifest)
	}))
	defer server.Close()
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/versions.json"
	source, err := NewSource(&settings)
	require.NoError(t, err)
	for _, test := range []struct {
		channel           toolchain.Channel
		selector, version string
	}{{toolchain.STS, "1.2", "1.2.10"}, {toolchain.UnknownChannel, "1.2", "1.2.10"}, {toolchain.Nightly, "2026-10-04", "1.3.0-alpha.20261004001050"}, {toolchain.UnknownChannel, "2026-10-04", "1.3.0-alpha.20261004001050"}} {
		resolved, err := source.ResolveToolchain(t.Context(), test.channel, test.selector, "linux-x64")
		require.NoError(t, err)
		require.Equal(t, test.version, resolved.Version)
	}
}
