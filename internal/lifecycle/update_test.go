package lifecycle

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/progress"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for UpdateInstalled and UpdateAll: which installed toolchain an
// update picks, and how the outcome is reported.

func updateHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv(config.EnvDistServer, "")
	require.NoError(t, config.EnsureDirs())
	return home
}

func saveUpdateSettings(t *testing.T, home string, settings config.Settings) {
	t.Helper()
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))
}

func fakeInstalled(t *testing.T, home string, names ...string) {
	t.Helper()
	for _, name := range names {
		require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", name), 0o755))
	}
}

func parse(t *testing.T, input string) toolchain.ToolchainName {
	t.Helper()
	name, err := toolchain.ParseToolchainName(input)
	require.NoError(t, err)
	return name
}

// manifestServer serves the manifest built from the server URL at
// /sdk-versions.json and the mock SDK archive for every /download/ path.
func manifestServer(t *testing.T, build func(base string) dist.Manifest) *httptest.Server {
	t.Helper()
	sdk, _ := testutil.CreateMockSDKZip("1.0.5")
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	manifest := build(server.URL)
	mux.HandleFunc("/download/", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(sdk)
	})
	mux.HandleFunc("/sdk-versions.json", func(w http.ResponseWriter, _ *http.Request) {
		assert.NoError(t, json.NewEncoder(w).Encode(manifest))
	})
	return server
}

// targetSDKServer publishes channel at version for the host and the given
// target tuples, so a target variant can be resolved and installed.
func targetSDKServer(t *testing.T, channel toolchain.Channel, version string, targets ...string) *httptest.Server {
	t.Helper()
	_, sha := testutil.CreateMockSDKZip("1.0.5")
	hostKey, err := sdktarget.CurrentHostTuple("")
	require.NoError(t, err)
	keys := []string{hostKey}
	for _, target := range targets {
		key, err := sdktarget.CurrentTargetTuple("", target)
		require.NoError(t, err)
		keys = append(keys, key)
	}
	return manifestServer(t, func(base string) dist.Manifest {
		channelInfo := func(version string) dist.ChannelInfo {
			platforms := make(map[string]dist.DownloadInfo)
			for _, key := range keys {
				name := "sdk-" + key + "-" + version + ".zip"
				platforms[key] = dist.DownloadInfo{Name: name, SHA256: sha, URL: base + "/download/" + name}
			}
			return dist.ChannelInfo{Latest: version, Versions: map[string]map[string]dist.DownloadInfo{version: platforms}}
		}
		var manifest dist.Manifest
		manifest.Channels.LTS, manifest.Channels.STS = channelInfo("1.0.5"), channelInfo("2.0.0")
		if channel == toolchain.LTS {
			manifest.Channels.LTS = channelInfo(version)
		} else {
			manifest.Channels.STS = channelInfo(version)
		}
		return manifest
	})
}

// splitNightlyServer mimics a dist_server root whose nightly channel lives in
// a separate nightly.json with URLs relative to the root.
func splitNightlyServer(t *testing.T) *httptest.Server {
	t.Helper()
	sdk, sha := testutil.CreateMockSDKZip("1.2.0")
	tuple, err := sdktarget.CurrentHostTuple("")
	require.NoError(t, err)
	const latest = "1.2.0-alpha.20260822010101"
	nightly := dist.ChannelInfo{
		Latest:   latest,
		Versions: map[string]map[string]dist.DownloadInfo{latest: {tuple: {Name: "nightly.zip", SHA256: sha, URL: "nightly/nightly.zip"}}},
	}
	manifest := dist.Manifest{}
	manifest.Channels.LTS = dist.ChannelInfo{Latest: "1.0.5", Versions: map[string]map[string]dist.DownloadInfo{"1.0.5": {tuple: {Name: "lts.zip", SHA256: sha, URL: "sdk/lts.zip"}}}}
	manifest.Channels.STS = dist.ChannelInfo{Latest: "1.1.0", Versions: map[string]map[string]dist.DownloadInfo{"1.1.0": {tuple: {Name: "sts.zip", SHA256: sha, URL: "sdk/sts.zip"}}}}
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	mux.HandleFunc("/corp/cjv/versions.json", func(w http.ResponseWriter, _ *http.Request) {
		assert.NoError(t, json.NewEncoder(w).Encode(manifest))
	})
	mux.HandleFunc("/corp/cjv/nightly.json", func(w http.ResponseWriter, _ *http.Request) {
		assert.NoError(t, json.NewEncoder(w).Encode(nightly))
	})
	mux.HandleFunc("/corp/cjv/nightly/nightly.zip", func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(sdk)
	})
	return server
}

func TestUpdateAllNoToolchains(t *testing.T) {
	updateHome(t)
	report, err := UpdateAll(t.Context(), quietLifecycleOptions())
	require.NoError(t, err, "no toolchains should be a no-op")
	assert.True(t, report.NoneInstalled)
	assert.Empty(t, report.Outcomes)
}

func TestUpdateAllReportsUpToDateAndSkipsCustom(t *testing.T) {
	home := updateHome(t)
	server := testutil.MockDistServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	saveUpdateSettings(t, home, settings)
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "lts"}, quietLifecycleOptions()))
	fakeInstalled(t, home, "local-sdk")

	report, err := UpdateAll(t.Context(), quietLifecycleOptions())
	require.NoError(t, err)
	assert.False(t, report.NoneInstalled)
	assert.Empty(t, applied(report))
	assert.ElementsMatch(t, []UpdateOutcome{
		{Name: "local-sdk", Status: UpdateSkipped},
		{Name: "lts-1.0.5", Replacement: "lts-1.0.5", Status: UpdateUpToDate},
	}, report.Outcomes)
}

func TestUpdateAllUpgradesReferencesAcrossToolchains(t *testing.T) {
	home := updateHome(t)
	fakeInstalled(t, home, "lts-1.0.0", "lts-1.0.5")
	server := testutil.MockDistServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	settings.DefaultToolchain = "lts-1.0.0"
	settings.Overrides[filepath.Join(home, "project")] = "lts-1.0.0"
	saveUpdateSettings(t, home, settings)

	report, err := UpdateAll(t.Context(), quietLifecycleOptions())
	require.NoError(t, err)
	assert.Equal(t, []UpdateOutcome{{Name: "lts-1.0.0", Replacement: "lts-1.0.5", Status: UpdateApplied}}, applied(report))
	loaded, err := config.LoadSettings(filepath.Join(home, ".cjv", "settings.toml"))
	require.NoError(t, err)
	assert.Equal(t, "lts-1.0.5", loaded.DefaultToolchain)
	assert.Equal(t, "lts-1.0.5", loaded.Overrides[filepath.Join(home, "project")])
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "lts-1.0.0"))
}

func TestUpdateAllContinuesAfterFailure(t *testing.T) {
	home := updateHome(t)
	// The manifest has no nightly channel, so the nightly toolchain fails to
	// resolve while lts still upgrades.
	fakeInstalled(t, home, "lts-1.0.0", "nightly-1.0.0-alpha.20260101000000")
	server := testutil.MockDistServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	saveUpdateSettings(t, home, settings)

	report, err := UpdateAll(t.Context(), quietLifecycleOptions())
	require.Error(t, err)
	assert.Len(t, report.Outcomes, 2)
	assert.Equal(t, []UpdateOutcome{{Name: "lts-1.0.0", Replacement: "lts-1.0.5", Status: UpdateApplied}}, applied(report))
	for _, o := range report.Outcomes {
		if o.Name != "lts-1.0.0" {
			assert.Equal(t, UpdateFailed, o.Status)
			assert.Error(t, o.Err)
		}
	}
}

func TestUpdateAllPurgesDownloadsDir(t *testing.T) {
	home := updateHome(t)
	leftover := filepath.Join(home, "downloads", "partial.zip")
	require.NoError(t, os.WriteFile(leftover, []byte("data"), 0o644))
	_, err := UpdateAll(t.Context(), quietLifecycleOptions())
	require.NoError(t, err)
	assert.NoFileExists(t, leftover)
}

func TestUpdateInstalledChannelAlreadyUpToDate(t *testing.T) {
	home := updateHome(t)
	server := testutil.MockDistServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	saveUpdateSettings(t, home, settings)
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "lts"}, quietLifecycleOptions()))

	recorder := &testutil.ProgressRecorder{}
	outcome, err := UpdateInstalled(t.Context(), parse(t, "lts"), Options{Progress: recorder})
	require.NoError(t, err)
	assert.Equal(t, UpdateOutcome{Name: "lts-1.0.5", Replacement: "lts-1.0.5", Status: UpdateUpToDate}, outcome)
	assert.Contains(t, recorder.Kinds, progress.AlreadyUpToDate)
}

func TestUpdateInstalledChannelUpgradesNewestInstalledVersion(t *testing.T) {
	home := updateHome(t)
	fakeInstalled(t, home, "lts-1.0.0", "sts-2.0.0")
	server := testutil.MockDistServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	settings.DefaultToolchain = "lts-1.0.0"
	settings.Overrides["C:\\project-a"] = "lts-1.0.0"
	settings.Overrides["C:\\project-b"] = "sts-2.0.0" // different channel, keep
	saveUpdateSettings(t, home, settings)

	outcome, err := UpdateInstalled(t.Context(), parse(t, "lts"), quietLifecycleOptions())
	require.NoError(t, err)
	assert.Equal(t, UpdateOutcome{Name: "lts-1.0.0", Replacement: "lts-1.0.5", Status: UpdateApplied}, outcome)

	installed, err := toolchain.ListInstalled()
	require.NoError(t, err)
	assert.Contains(t, installed, "lts-1.0.5")
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "lts-1.0.0"), "old version directory should be removed")

	reloaded, err := config.LoadSettings(filepath.Join(home, ".cjv", "settings.toml"))
	require.NoError(t, err)
	assert.Equal(t, "lts-1.0.5", reloaded.DefaultToolchain, "default should be updated from old version to new version")
	assert.Equal(t, "lts-1.0.5", reloaded.Overrides["C:\\project-a"], "override should be updated to new version")
	assert.Equal(t, "sts-2.0.0", reloaded.Overrides["C:\\project-b"], "unrelated override should be preserved")
}

func TestUpdateInstalledNightlyUsesUnifiedDistServer(t *testing.T) {
	home := updateHome(t)
	oldName := "nightly-1.1.0-alpha.20260821010101"
	fakeInstalled(t, home, oldName)
	server := splitNightlyServer(t)
	settings := config.DefaultSettings()
	settings.DistServer = server.URL + "/corp/cjv"
	saveUpdateSettings(t, home, settings)

	outcome, err := UpdateInstalled(t.Context(), parse(t, "nightly"), quietLifecycleOptions())
	require.NoError(t, err)
	assert.Equal(t, UpdateOutcome{Name: oldName, Replacement: "nightly-1.2.0-alpha.20260822010101", Status: UpdateApplied}, outcome)
}

func TestUpdateInstalledTargetVariantUpdatesVariant(t *testing.T) {
	home := updateHome(t)
	targetKey, err := sdktarget.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	oldName := "sts-1.0.0-" + targetKey
	fakeInstalled(t, home, oldName)
	server := targetSDKServer(t, toolchain.STS, "2.0.0", "ohos")
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	saveUpdateSettings(t, home, settings)

	outcome, err := UpdateInstalled(t.Context(), parse(t, oldName), quietLifecycleOptions())
	require.NoError(t, err)
	assert.Equal(t, UpdateOutcome{Name: oldName, Replacement: "sts-2.0.0-" + targetKey, Status: UpdateApplied}, outcome)

	installed, err := toolchain.ListInstalled()
	require.NoError(t, err)
	assert.Contains(t, installed, "sts-2.0.0-"+targetKey)
	assert.NotContains(t, installed, "sts-2.0.0", "the host variant must not be installed alongside")
	assert.NotContains(t, installed, oldName)
}

func TestUpdateInstalledExplicitVersionInstallsWhenMissing(t *testing.T) {
	home := updateHome(t)
	server := testutil.MockDistServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	saveUpdateSettings(t, home, settings)

	outcome, err := UpdateInstalled(t.Context(), parse(t, "lts-1.0.5"), quietLifecycleOptions())
	require.NoError(t, err)
	assert.Equal(t, UpdateOutcome{Name: "lts-1.0.5", Status: UpdatePinned}, outcome)
	assert.DirExists(t, filepath.Join(home, "toolchains", "lts-1.0.5"))

	// Already installed: still a no-op success.
	outcome, err = UpdateInstalled(t.Context(), parse(t, "lts-1.0.5"), quietLifecycleOptions())
	require.NoError(t, err)
	assert.Equal(t, UpdatePinned, outcome.Status)
}

func TestUpdateInstalledRejectsCustomMissingChannelAndMissingVariant(t *testing.T) {
	home := updateHome(t)
	saveUpdateSettings(t, home, config.DefaultSettings())

	_, err := UpdateInstalled(t.Context(), parse(t, "local-sdk"), quietLifecycleOptions())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "local-sdk")

	_, err = UpdateInstalled(t.Context(), parse(t, "lts"), quietLifecycleOptions())
	var notInstalled *cjverr.ToolchainNotInstalledError
	require.ErrorAs(t, err, &notInstalled)
	assert.Equal(t, "lts", notInstalled.Name)

	targetKey, err := sdktarget.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	_, err = UpdateInstalled(t.Context(), parse(t, "sts-2.0.0-"+targetKey), quietLifecycleOptions())
	require.ErrorAs(t, err, &notInstalled)
	assert.Equal(t, "sts-2.0.0-"+targetKey, notInstalled.Name)
}

// installedForChannel picks the version a channel-only update replaces.

func TestInstalledForChannelFindsLatest(t *testing.T) {
	home := updateHome(t)
	fakeInstalled(t, home, "lts-1.0.0", "lts-1.0.5", "sts-2.0.0")

	name, err := installedForChannel(toolchain.LTS)
	require.NoError(t, err)
	assert.Equal(t, "lts-1.0.5", name, "should return the latest semver version for the channel")
}

func TestInstalledForChannelNightly(t *testing.T) {
	home := updateHome(t)
	fakeInstalled(t, home, "nightly-20260301")

	name, err := installedForChannel(toolchain.Nightly)
	require.NoError(t, err)
	assert.Equal(t, "nightly-20260301", name)
}

func TestInstalledForChannelIgnoresTargetVariants(t *testing.T) {
	home := updateHome(t)
	targetKey, err := sdktarget.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	fakeInstalled(t, home, "lts-1.0.5-"+targetKey)

	name, err := installedForChannel(toolchain.LTS)
	require.NoError(t, err)
	assert.Empty(t, name)
}

func TestInstalledForChannelNotInstalled(t *testing.T) {
	home := updateHome(t)
	fakeInstalled(t, home, "lts-1.0.5")

	name, err := installedForChannel(toolchain.STS)
	require.NoError(t, err)
	assert.Empty(t, name, "should return empty string when channel has no installs")

	require.NoError(t, os.RemoveAll(filepath.Join(home, "toolchains")))
	name, err = installedForChannel(toolchain.LTS)
	require.NoError(t, err)
	assert.Empty(t, name)
}

func applied(r UpdateReport) []UpdateOutcome {
	var out []UpdateOutcome
	for _, o := range r.Outcomes {
		if o.Status == UpdateApplied {
			out = append(out, o)
		}
	}
	return out
}
