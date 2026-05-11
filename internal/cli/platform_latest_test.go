package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/cli/output"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func manifestOnlyServer(t *testing.T, manifest dist.Manifest) *httptest.Server {
	t.Helper()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	mux.HandleFunc("/sdk-versions.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(manifest); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	return server
}

func manifestWithPlatformGap() dist.Manifest {
	const sha = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	var manifest dist.Manifest
	manifest.Channels.LTS = dist.ChannelInfo{
		Latest: "2.0.0",
		Versions: map[string]map[string]dist.DownloadInfo{
			"2.0.0": {
				"win32-x64": {
					Name:   "cangjie-sdk-win32-x64-2.0.0.zip",
					SHA256: sha,
					URL:    "https://example.invalid/cangjie-sdk-win32-x64-2.0.0.zip",
				},
			},
			"1.5.0": {
				"linux-x64": {
					Name:   "cangjie-sdk-linux-x64-1.5.0.tar.gz",
					SHA256: sha,
					URL:    "https://example.invalid/cangjie-sdk-linux-x64-1.5.0.tar.gz",
				},
				"linux-x64-ohos": {
					Name:   "cangjie-sdk-linux-x64-ohos-1.5.0.tar.gz",
					SHA256: sha,
					URL:    "https://example.invalid/cangjie-sdk-linux-x64-ohos-1.5.0.tar.gz",
				},
			},
		},
	}
	manifest.Channels.STS = dist.ChannelInfo{
		Latest: "3.0.0",
		Versions: map[string]map[string]dist.DownloadInfo{
			"3.0.0": {
				"linux-x64": {
					Name:   "cangjie-sdk-linux-x64-3.0.0.tar.gz",
					SHA256: sha,
					URL:    "https://example.invalid/cangjie-sdk-linux-x64-3.0.0.tar.gz",
				},
			},
		},
	}
	return manifest
}

func TestResolveAndLocateWithTupleUsesLatestVersionAvailableForTuple(t *testing.T) {
	server := manifestOnlyServer(t, manifestWithPlatformGap())
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"

	resolved, err := resolveAndLocateWithTuple(context.Background(), toolchain.ToolchainName{
		Channel: toolchain.LTS,
	}, &settings, newManifestFetcher(settings.ManifestURL), "linux-x64")

	require.NoError(t, err)
	assert.Equal(t, "lts-1.5.0", resolved.Name)
	assert.Equal(t, "cangjie-sdk-linux-x64-1.5.0.tar.gz", resolved.ArchiveName)
}

func TestRunCheckUsesLatestVersionAvailableForInstalledTarget(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "lts-1.0.0-linux-x64-ohos"), 0o755))

	server := manifestOnlyServer(t, manifestWithPlatformGap())
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	output.SetJSONMode(true)
	t.Cleanup(func() { output.SetJSONMode(false) })

	var buf bytes.Buffer
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	cmd.SetOut(&buf)
	require.NoError(t, runCheck(cmd, nil))

	var got checkResult
	require.NoError(t, json.Unmarshal(buf.Bytes(), &got))
	require.Len(t, got.Toolchains, 1)
	assert.Equal(t, "lts-1.5.0-linux-x64-ohos", got.Toolchains[0].Latest)
	assert.True(t, got.Toolchains[0].UpdateAvailable)
	assert.False(t, got.Toolchains[0].NotForTarget)
}
