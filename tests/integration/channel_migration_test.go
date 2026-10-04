//go:build integration

package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Zxilly/cjv/internal/dist"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIntegrationLegacyMigrationAndUpgradeKeepPinnedSDK(t *testing.T) {
	binary := buildCJV(t)
	archive, checksum := createExecutableMockSDKZip(t, buildStubExecutable(t))
	tuple, err := sdktarget.CurrentHostTuple("")
	require.NoError(t, err)
	var manifestRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/sdk-versions.json" {
			_, _ = w.Write(archive)
			return
		}
		manifestRequests.Add(1)
		var manifest dist.Manifest
		info := dist.ChannelInfo{Latest: "1.0.6", Versions: make(map[string]map[string]dist.DownloadInfo)}
		for _, version := range []string{"1.0.5", "1.0.6"} {
			info.Versions[version] = map[string]dist.DownloadInfo{tuple: {
				Name: "sdk-" + version + ".zip", SHA256: checksum,
				URL: "http://" + r.Host + "/sdk-" + version + ".zip",
			}}
		}
		manifest.Channels.LTS, manifest.Channels.STS = info, info
		_ = json.NewEncoder(w).Encode(manifest)
	}))
	t.Cleanup(server.Close)
	home := setupIntegrationEnv(t, binary, server.URL)
	settingsPath := filepath.Join(home, ".cjv", "settings.toml")
	settings := fmt.Sprintf("manifest_url = %q\ndefault_toolchain = \"lts-1.0.5\"\nauto_self_update = \"disable\"\n", server.URL+"/sdk-versions.json")
	require.NoError(t, os.WriteFile(settingsPath, []byte(settings), 0o644))
	legacyDir := filepath.Join(home, "toolchains", "lts-1.0.5")
	archivePath := filepath.Join(t.TempDir(), "legacy.zip")
	require.NoError(t, os.WriteFile(archivePath, archive, 0o644))
	require.NoError(t, dist.InstallSDK(t.Context(), archivePath, legacyDir))
	require.NoError(t, os.WriteFile(filepath.Join(legacyDir, "user.txt"), []byte("keep legacy data"), 0o644))

	stdout, stderr, err := runCJV(t, binary, home, "show", "active")
	require.NoError(t, err, "migration failed: stdout=%s stderr=%s", stdout, stderr)
	assert.Contains(t, stdout, "lts-1.0.5")
	assert.Zero(t, manifestRequests.Load(), "migration must finish offline")
	persisted, err := os.ReadFile(settingsPath)
	require.NoError(t, err)
	assert.Equal(t, settings, string(persisted), "migration must not rewrite selectors")
	channelDir := filepath.Join(home, "toolchains", "lts")
	record, err := toolchain.ReadInstallation(channelDir)
	require.NoError(t, err)
	assert.Equal(t, "lts-1.0.5", record.Release)
	old, err := os.Stat(filepath.Join(legacyDir, "user.txt"))
	require.NoError(t, err)
	copied, err := os.Stat(filepath.Join(channelDir, "user.txt"))
	require.NoError(t, err)
	assert.False(t, os.SameFile(old, copied), "legacy migration copies rather than shares files")

	stdout, stderr, err = runCJV(t, binary, home, "update", "lts", "--no-self-update")
	require.NoError(t, err, "update failed: stdout=%s stderr=%s", stdout, stderr)
	record, err = toolchain.ReadInstallation(channelDir)
	require.NoError(t, err)
	assert.Equal(t, "lts-1.0.6", record.Release)
	assert.NoFileExists(t, filepath.Join(channelDir, "user.txt"))
	assert.FileExists(t, filepath.Join(legacyDir, "user.txt"))
	stdout, _, err = runCJV(t, binary, home, "show", "active")
	require.NoError(t, err)
	assert.Contains(t, stdout, "lts-1.0.5", "the old explicit default must remain fixed")

	stdout, stderr, err = runCJV(t, binary, home, "uninstall", "lts")
	require.NoError(t, err, "uninstall failed: stdout=%s stderr=%s", stdout, stderr)
	stdout, _, err = runCJV(t, binary, home, "toolchain", "list")
	require.NoError(t, err)
	assert.Contains(t, stdout, "lts-1.0.5")
	assert.NotContains(t, strings.ReplaceAll(stdout, "\r\n", "\n"), "  lts\n")
	assert.NoDirExists(t, channelDir, "a completed migration must not resurrect the channel")
	assert.FileExists(t, filepath.Join(legacyDir, "user.txt"))
}
