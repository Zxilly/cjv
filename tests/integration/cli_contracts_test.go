//go:build integration

package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func decodeCLIJSON(t *testing.T, stdout string) map[string]any {
	t.Helper()
	decoder := json.NewDecoder(strings.NewReader(stdout))
	var result map[string]any
	require.NoError(t, decoder.Decode(&result), stdout)
	require.ErrorIs(t, decoder.Decode(new(any)), io.EOF, "stdout contains one JSON document")
	return result
}

func TestIntegrationSettingsAndOverrideJSON(t *testing.T) {
	binary, home, project := buildCJV(t), t.TempDir(), t.TempDir()
	run := func(args ...string) map[string]any {
		t.Helper()
		stdout, stderr, err := runCJV(t, binary, home, append(args, "--json")...)
		require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
		return decodeCLIJSON(t, stdout)
	}
	for _, changed := range []bool{true, false} {
		result := run("set", "auto-install", "false")
		assert.Equal(t, "auto-install", result["key"])
		assert.Equal(t, false, result["value"])
		assert.Equal(t, changed, result["changed"])
	}
	assert.Equal(t, []any{}, run("override", "list")["overrides"])
	result := run("override", "set", "lts-1.0.5", "--path", project)
	assert.Equal(t, config.NormalizePath(project), result["directory"])
	assert.Equal(t, "lts-1.0.5", result["toolchain"])
	assert.Equal(t, []any{map[string]any{"directory": config.NormalizePath(project), "toolchain": "lts-1.0.5"}}, run("override", "list")["overrides"])
	assert.Equal(t, []any{config.NormalizePath(project)}, run("override", "unset", "--path", project)["removed"])
	assert.Equal(t, []any{}, run("override", "unset", "--nonexistent")["removed"])
}

func TestIntegrationRootAndShowShareStatus(t *testing.T) {
	binary := buildCJV(t)
	server := testutil.MockDistServer(t)
	home := setupIntegrationEnv(t, binary, server.URL)
	for _, installed := range []bool{false, true} {
		if installed {
			stdout, stderr, err := runCJV(t, binary, home, "install", "lts")
			require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
		}
		for _, jsonMode := range []bool{false, true} {
			var args []string
			if jsonMode {
				args = append(args, "--json")
			}
			root, rootErr, err := runCJV(t, binary, home, args...)
			require.NoError(t, err, "stderr=%s", rootErr)
			show, showErr, err := runCJV(t, binary, home, append([]string{"show"}, args...)...)
			require.NoError(t, err, "stderr=%s", showErr)
			assert.Equal(t, root, show)
			assert.Equal(t, rootErr, showErr)
			if jsonMode {
				decodeCLIJSON(t, root)
			}
		}
	}
}

func TestIntegrationUninstallRequiresExplicitConfirmation(t *testing.T) {
	binary := buildCJV(t)
	server := testutil.MockDistServer(t)
	home := setupIntegrationEnv(t, binary, server.URL)
	stdout, stderr, err := runCJV(t, binary, home, "install", "lts")
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
	dir := filepath.Join(home, "toolchains", "lts")
	stdout, _, err = runCJV(t, binary, home, "uninstall", "lts", "--json")
	require.Error(t, err)
	assert.Contains(t, decodeCLIJSON(t, stdout), "error")
	assert.DirExists(t, dir)
	_, stderr, err = runCJV(t, binary, home, "uninstall", "lts")
	require.Error(t, err)
	assert.Contains(t, stderr, "--yes")
	assert.DirExists(t, dir)
	stdout, stderr, err = runCJV(t, binary, home, "uninstall", "lts", "--yes", "--json")
	require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
	assert.Equal(t, "lts", decodeCLIJSON(t, stdout)["name"])
	assert.NoDirExists(t, dir)
}

func TestIntegrationTrackingTargetOwnership(t *testing.T) {
	binary := buildCJV(t)
	archive, checksum := testutil.CreateMockSDKZip("1.2.0")
	hostTuple, err := sdktarget.CurrentHostTuple("")
	require.NoError(t, err)
	targetTuple, err := sdktarget.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	for _, operation := range []string{"combined-install", "target-add"} {
		t.Run(operation, func(t *testing.T) {
			var downloads atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/sdk.zip" {
					downloads.Add(1)
					_, _ = w.Write(archive)
					return
				}
				var manifest dist.Manifest
				artifact := dist.DownloadInfo{Name: "sdk.zip", SHA256: checksum, URL: "http://" + r.Host + "/sdk.zip"}
				manifest.Channels.STS = dist.ChannelInfo{Latest: "1.2.0", Versions: map[string]map[string]dist.DownloadInfo{
					"1.2.0": {hostTuple: artifact, targetTuple: artifact},
				}}
				manifest.Channels.LTS = manifest.Channels.STS
				_ = json.NewEncoder(w).Encode(manifest)
			}))
			t.Cleanup(server.Close)
			home := setupIntegrationEnv(t, binary, server.URL)
			run := func(args ...string) map[string]any {
				t.Helper()
				stdout, stderr, err := runCJV(t, binary, home, append(args, "--json")...)
				require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
				return decodeCLIJSON(t, stdout)
			}
			if operation == "combined-install" {
				run("install", "sts", "--target", "ohos")
			} else {
				run("install", "sts")
				run("target", "add", "ohos", "--toolchain", "sts")
			}
			verifyOwnership := func() {
				t.Helper()
				targets := run("target", "list", "--installed", "--toolchain", "sts")
				assert.Equal(t, "sts", targets["toolchain"])
				assert.Equal(t, []any{map[string]any{"name": "ohos", "installed": true}}, targets["targets"])
				installed := run("toolchain", "list")["toolchains"].([]any)
				var names []string
				for _, raw := range installed {
					entry := raw.(map[string]any)
					names = append(names, entry["name"].(string))
					assert.Equal(t, "tracking", entry["kind"])
				}
				assert.ElementsMatch(t, []string{"sts", "sts-" + targetTuple}, names, "targets belong to the tracking host; no fixed-version target is materialized")
			}
			verifyOwnership()
			beforeRepeat := downloads.Load()
			run("install", "sts", "--target", "ohos")
			run("target", "add", "ohos", "--toolchain", "sts")
			verifyOwnership()
			assert.Equal(t, beforeRepeat, downloads.Load(), "repeating either install path reuses the installed SDKs")
			run("uninstall", "sts", "--yes")
			remaining := run("toolchain", "list")
			require.Contains(t, remaining, "toolchains")
			assert.Equal(t, []any{}, remaining["toolchains"])
			assert.NoDirExists(t, filepath.Join(home, "toolchains", "sts"))
			assert.NoDirExists(t, filepath.Join(home, "toolchains", "sts-"+targetTuple))
		})
	}
}
