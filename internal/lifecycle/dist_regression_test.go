package lifecycle_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/lifecycle"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/require"
)

func TestUnchangedUpdatePreservesLegacyChecksumCasing(t *testing.T) {
	for _, legacyRecord := range []bool{false, true} {
		name := "normalized-installation"
		if legacyRecord {
			name = "legacy-uppercase-installation"
		}
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			configtest.Isolate(t, home)
			t.Setenv(config.EnvNoPathSetup, "1")
			t.Setenv(config.EnvDistServer, "")
			t.Setenv("CJV_OFFLINE", "")
			archive, digest := testutil.MockSDKZip()
			server := testutil.MockServerWithSDK(t, archive, strings.ToUpper(digest))
			settings := config.DefaultSettings()
			settings.ManifestURL = server.URL + "/sdk-versions.json"
			settingsPath, err := config.SettingsPath()
			require.NoError(t, err)
			require.NoError(t, config.SaveSettings(&settings, settingsPath))
			require.NoError(t, lifecycle.Install(t.Context(), lifecycle.InstallRequest{Toolchain: "lts"}, lifecycle.Options{}))
			directory := filepath.Join(home, config.ToolchainsSubdir, "lts")
			record, err := toolchain.ReadInstallation(directory)
			require.NoError(t, err)
			if legacyRecord {
				record.SHA256 = strings.ToUpper(digest)
				require.NoError(t, toolchain.WriteInstallation(directory, record))
			}
			marker := filepath.Join(directory, "local-no-change-marker")
			require.NoError(t, os.WriteFile(marker, []byte("preserve on unchanged update"), 0o600))
			identity := toolchain.ToolchainName{Channel: toolchain.LTS}
			plan, err := lifecycle.PlanUpdate(t.Context(), identity, lifecycle.Options{})
			require.NoError(t, err)
			require.False(t, plan.UpdateAvailable, "checksum case alone does not make an update available")
			require.Equal(t, "up-to-date", plan.Status)
			outcome, err := lifecycle.UpdateInstalled(t.Context(), identity, lifecycle.Options{})
			require.NoError(t, err)
			require.Equal(t, lifecycle.UpdateUpToDate, outcome.Status)
			entries, err := lifecycle.ListHistory(t.Context(), "lts")
			require.NoError(t, err)
			require.Empty(t, entries)
			require.FileExists(t, marker)
			after, err := toolchain.ReadInstallation(directory)
			require.NoError(t, err)
			require.Equal(t, record, after)
		})
	}
}

func TestOfflineReinstallUsesCachedUppercaseManifestDigest(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	t.Setenv(config.EnvNoPathSetup, "1")
	t.Setenv(config.EnvDistServer, "")
	t.Setenv("CJV_OFFLINE", "")
	archive, digest := testutil.MockSDKZip()
	digest = strings.ToUpper(digest)
	server := testutil.MockServerWithSDK(t, archive, digest)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	settingsPath, err := config.SettingsPath()
	require.NoError(t, err)
	require.NoError(t, config.SaveSettings(&settings, settingsPath))

	require.NoError(t, lifecycle.Install(t.Context(), lifecycle.InstallRequest{Toolchain: "lts"}, lifecycle.Options{}))
	require.NoFileExists(t, filepath.Join(home, "downloads", strings.ToLower(digest)))
	err = lifecycle.Install(dist.WithOffline(t.Context(), true), lifecycle.InstallRequest{Toolchain: "lts", Reinstall: true}, lifecycle.Options{})
	require.NoError(t, err, "the successful installation must leave a reusable archive even when the manifest uses uppercase hex")
}

func TestOfflineNightlyPlanUsesCachedMissingChecksum(t *testing.T) {
	configtest.Isolate(t, t.TempDir())
	t.Setenv(config.EnvDistServer, "")
	t.Setenv("CJV_OFFLINE", "")
	tuple, err := sdktarget.CurrentHostTuple("")
	require.NoError(t, err)
	const version = "1.2.0-alpha.20260101010101"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/nightly.json" {
			http.NotFound(w, r)
			return
		}
		_ = json.NewEncoder(w).Encode(dist.ChannelInfo{
			Latest: version,
			Versions: map[string]map[string]dist.DownloadInfo{
				version: {tuple: {Name: "nightly.zip", URL: "nightly.zip"}},
			},
		})
	}))
	defer server.Close()
	settings := config.DefaultSettings()
	settings.DistServer = server.URL
	settingsPath, err := config.SettingsPath()
	require.NoError(t, err)
	require.NoError(t, config.SaveSettings(&settings, settingsPath))
	source, err := dist.NewSource(&settings)
	require.NoError(t, err)
	_, err = source.ResolveToolchain(t.Context(), toolchain.Nightly, "", tuple)
	require.NoError(t, err)

	online, err := lifecycle.PlanUpdates(t.Context(), []string{"nightly"}, lifecycle.Options{})
	require.NoError(t, err)
	offline, err := lifecycle.PlanUpdates(dist.WithOffline(t.Context(), true), []string{"nightly"}, lifecycle.Options{})
	require.NoError(t, err)
	require.Equal(t, online, offline, "metadata-only planning needs no archive when the source explicitly has no checksum")
}

func TestUppercaseManifestChecksumsSupportHistoryRollback(t *testing.T) {
	for _, legacyRecord := range []bool{false, true} {
		name := "normalized-installation"
		if legacyRecord {
			name = "legacy-uppercase-installation"
		}
		t.Run(name, func(t *testing.T) {
			home := t.TempDir()
			configtest.Isolate(t, home)
			t.Setenv(config.EnvNoPathSetup, "1")
			t.Setenv(config.EnvDistServer, "")
			t.Setenv("CJV_OFFLINE", "")
			archive, digest := testutil.MockSDKZip()
			tuple, err := sdktarget.CurrentHostTuple("")
			require.NoError(t, err)
			var version atomic.Value
			version.Store("1.0.0")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/sdk.zip" {
					_, _ = w.Write(archive)
					return
				}
				current := version.Load().(string)
				channel := dist.ChannelInfo{Latest: current, Versions: map[string]map[string]dist.DownloadInfo{
					current: {tuple: {Name: "sdk.zip", URL: "sdk.zip", SHA256: strings.ToUpper(digest)}},
				}}
				manifest := dist.Manifest{}
				manifest.Channels.LTS, manifest.Channels.STS = channel, channel
				_ = json.NewEncoder(w).Encode(manifest)
			}))
			defer server.Close()
			settings := config.DefaultSettings()
			settings.DistServer = server.URL
			settingsPath, err := config.SettingsPath()
			require.NoError(t, err)
			require.NoError(t, config.SaveSettings(&settings, settingsPath))
			require.NoError(t, lifecycle.Install(t.Context(), lifecycle.InstallRequest{Toolchain: "lts"}, lifecycle.Options{}))
			directory := filepath.Join(home, config.ToolchainsSubdir, "lts")
			expected := digest
			if legacyRecord {
				record, err := toolchain.ReadInstallation(directory)
				require.NoError(t, err)
				record.SHA256 = strings.ToUpper(digest)
				expected = record.SHA256
				require.NoError(t, toolchain.WriteInstallation(directory, record))
			}
			version.Store("2.0.0")
			_, err = lifecycle.UpdateInstalled(t.Context(), toolchain.ToolchainName{Channel: toolchain.LTS}, lifecycle.Options{})
			require.NoError(t, err)
			history, err := lifecycle.ListHistory(t.Context(), "lts")
			require.NoError(t, err)
			require.Len(t, history, 1)
			require.Equal(t, expected, history[0].Members[0].SHA256)
			_, err = lifecycle.RollbackHistory(dist.WithOffline(t.Context(), true), "lts", history[0].ID, lifecycle.Options{})
			require.NoError(t, err)
			record, err := toolchain.ReadInstallation(directory)
			require.NoError(t, err)
			require.Equal(t, "lts-1.0.0", record.Release)
			require.Equal(t, expected, record.SHA256, "history preserves the installation record's original representation")
		})
	}
}
