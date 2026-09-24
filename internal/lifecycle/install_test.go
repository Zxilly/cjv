package lifecycle_test

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/progress"
	"github.com/Zxilly/cjv/internal/sdktools"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for Install: the full pipeline from a toolchain request through
// resolution, download, extraction, validation and the transactional swap,
// against a local distribution server.

// installHome isolates CJV_HOME and points the manifest at server, returning
// the home directory.
func installHome(t *testing.T, manifestURL string) string {
	t.Helper()
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv(config.EnvDistServer, "")
	require.NoError(t, config.EnsureDirs())
	settings := config.DefaultSettings()
	settings.ManifestURL = manifestURL
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))
	return home
}

// distServerHome isolates CJV_HOME with dist_server set to root.
func distServerHome(t *testing.T, root string) string {
	t.Helper()
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv(config.EnvDistServer, "")
	require.NoError(t, config.EnsureDirs())
	settings := config.DefaultSettings()
	settings.DistServer = root
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))
	return home
}

func install(t *testing.T, req lifecycle.InstallRequest) error {
	t.Helper()
	return lifecycle.Install(context.Background(), req, lifecycle.Options{})
}

func installedNames(t *testing.T) []string {
	t.Helper()
	installed, err := toolchain.ListInstalled()
	require.NoError(t, err)
	return installed
}

func TestInstall_ResolvesChannelsAndVersions(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"lts channel", "lts", "lts-1.0.5"},
		{"sts channel", "sts", "sts-2.0.0"},
		{"specific version", "lts-1.0.5", "lts-1.0.5"},
		// A bare version discovers its channel from the manifest.
		{"bare version lts", "1.0.5", "lts-1.0.5"},
		{"bare version sts", "2.0.0", "sts-2.0.0"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			installHome(t, testutil.ValidMockServer(t).URL+"/sdk-versions.json")
			require.NoError(t, install(t, lifecycle.InstallRequest{Toolchain: tt.input}))
			assert.Contains(t, installedNames(t), tt.want)
		})
	}
}

func TestInstall_RejectsUnknownVersionAndInvalidNames(t *testing.T) {
	installHome(t, testutil.ValidMockServer(t).URL+"/sdk-versions.json")
	assert.Error(t, install(t, lifecycle.InstallRequest{Toolchain: "99.99.99"}), "non-existent version should fail")
	assert.Error(t, install(t, lifecycle.InstallRequest{Toolchain: "+invalid"}), "invalid name starting with + should fail")

	err := install(t, lifecycle.InstallRequest{Toolchain: "local-sdk"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "local-sdk")
	assert.Contains(t, err.Error(), "cjv toolchain link")

	err = install(t, lifecycle.InstallRequest{Toolchain: "lts-1.0.5-linux-x64-ohos", Targets: []string{"android"}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot combine target variant")
	assert.Empty(t, installedNames(t))
}

func TestInstall_BothChannels(t *testing.T) {
	installHome(t, testutil.ValidMockServer(t).URL+"/sdk-versions.json")
	require.NoError(t, install(t, lifecycle.InstallRequest{Toolchain: "lts"}))
	require.NoError(t, install(t, lifecycle.InstallRequest{Toolchain: "sts"}))
	installed := installedNames(t)
	assert.Len(t, installed, 2)
	assert.Contains(t, installed, "lts-1.0.5")
	assert.Contains(t, installed, "sts-2.0.0")
}

func TestInstall_NightlyFromUnifiedDistServer(t *testing.T) {
	distServerHome(t, testutil.SplitNightlyMockServer(t).URL+"/corp/cjv")
	require.NoError(t, install(t, lifecycle.InstallRequest{Toolchain: "nightly"}))
	assert.Contains(t, installedNames(t), "nightly-1.2.0-alpha.20260822010101")
}

func TestInstall_DefaultManifestInstallsNightly(t *testing.T) {
	// nightly.json is found beside versions.json without dist_server.
	installHome(t, testutil.SplitNightlyMockServer(t).URL+"/corp/cjv/versions.json")
	require.NoError(t, install(t, lifecycle.InstallRequest{Toolchain: "nightly"}))
	assert.Contains(t, installedNames(t), "nightly-1.2.0-alpha.20260822010101")
}

func TestInstall_NightlyComponentFromUnifiedDistServer(t *testing.T) {
	home := distServerHome(t, testutil.SplitNightlyMockServer(t).URL+"/corp/cjv")
	require.NoError(t, install(t, lifecycle.InstallRequest{Toolchain: "nightly", Components: []string{"docs"}}))
	assert.FileExists(t, filepath.Join(home, "docs", "nightly-1.2.0-alpha.20260822010101", "main", "index.html"))
}

func TestInstall_AlreadyInstalledAndForce(t *testing.T) {
	installHome(t, testutil.ValidMockServer(t).URL+"/sdk-versions.json")
	require.NoError(t, install(t, lifecycle.InstallRequest{Toolchain: "lts"}))

	recorder := &testutil.ProgressRecorder{}
	opts := lifecycle.Options{Progress: recorder}
	// Second install without force reports "already installed" and returns nil.
	assert.NoError(t, lifecycle.Install(context.Background(), lifecycle.InstallRequest{Toolchain: "lts"}, opts),
		"already-installed is an informational no-op, not an error")
	assert.Equal(t, []progress.Kind{progress.FetchingManifest, progress.ToolchainAlreadyInstalled}, recorder.Kinds)

	assert.NoError(t, install(t, lifecycle.InstallRequest{Toolchain: "lts", Force: true}),
		"force install should succeed even when already installed")
}

func TestInstall_FirstInstallSetsDefaultAndBootstrapsManagedBinary(t *testing.T) {
	home := installHome(t, testutil.ValidMockServer(t).URL+"/sdk-versions.json")
	managedBinary := filepath.Join(home, "bin", sdktools.CjvBinaryName())
	require.NoFileExists(t, managedBinary)

	require.NoError(t, install(t, lifecycle.InstallRequest{Toolchain: "lts"}))

	reloaded, err := config.LoadSettings(filepath.Join(home, ".cjv", "settings.toml"))
	require.NoError(t, err)
	assert.Equal(t, "lts-1.0.5", reloaded.DefaultToolchain, "first install should set the default toolchain")
	assert.FileExists(t, managedBinary, "first install should bootstrap the managed cjv binary")
}

func TestInstall_FailsWhenCompilerMissing(t *testing.T) {
	// An archive without bin/cjc extracts fine but fails validation, so
	// nothing is placed and the staging tree is cleaned up.
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	f, err := w.Create("cangjie/tools/bin/" + sdktools.PlatformBinaryName("cjpm"))
	require.NoError(t, err)
	_, err = f.Write([]byte("stub"))
	require.NoError(t, err)
	require.NoError(t, w.Close())
	sum := sha256.Sum256(buf.Bytes())
	home := installHome(t, testutil.MockServerWithSDK(t, buf.Bytes(), hex.EncodeToString(sum[:])).URL+"/sdk-versions.json")

	err = install(t, lifecycle.InstallRequest{Toolchain: "lts"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "installation validation failed")
	entries, err := os.ReadDir(filepath.Join(home, "toolchains"))
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestInstall_HostAndTargets(t *testing.T) {
	home := installHome(t, testutil.MockServerWithTargetSDKs(t, toolchain.STS, "2.0.0", "ohos", "android").URL+"/sdk-versions.json")
	require.NoError(t, install(t, lifecycle.InstallRequest{Toolchain: "sts", Targets: []string{"ohos", "android"}}))

	hostKey, err := sdktarget.CurrentHostTuple("")
	require.NoError(t, err)
	ohosKey, err := sdktarget.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	androidKey, err := sdktarget.CurrentTargetTuple("", "android")
	require.NoError(t, err)

	installed := installedNames(t)
	assert.Contains(t, installed, "sts-2.0.0")
	assert.NotContains(t, installed, "sts-2.0.0-"+hostKey)
	assert.Contains(t, installed, "sts-2.0.0-"+ohosKey)
	assert.Contains(t, installed, "sts-2.0.0-"+androidKey)

	reloaded, err := config.LoadSettings(filepath.Join(home, ".cjv", "settings.toml"))
	require.NoError(t, err)
	assert.Equal(t, "sts-2.0.0", reloaded.DefaultToolchain)
}

func TestInstall_BareVersionWithTargetsResolvesChannel(t *testing.T) {
	installHome(t, testutil.MockServerWithTargetSDKs(t, toolchain.STS, "2.0.0", "ohos").URL+"/sdk-versions.json")
	require.NoError(t, install(t, lifecycle.InstallRequest{Toolchain: "2.0.0", Targets: []string{"ohos"}}))

	ohosKey, err := sdktarget.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	installed := installedNames(t)
	assert.Contains(t, installed, "sts-2.0.0")
	assert.Contains(t, installed, "sts-2.0.0-"+ohosKey)
}

func TestInstall_ExplicitVariantDoesNotSetDefault(t *testing.T) {
	home := installHome(t, testutil.MockServerWithTargetSDKs(t, toolchain.STS, "2.0.0", "ohos").URL+"/sdk-versions.json")
	ohosKey, err := sdktarget.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	require.NoError(t, install(t, lifecycle.InstallRequest{Toolchain: "sts-2.0.0-" + ohosKey}))

	installed := installedNames(t)
	assert.Contains(t, installed, "sts-2.0.0-"+ohosKey)
	assert.NotContains(t, installed, "sts-2.0.0")

	reloaded, err := config.LoadSettings(filepath.Join(home, ".cjv", "settings.toml"))
	require.NoError(t, err)
	assert.Empty(t, reloaded.DefaultToolchain)
}

// manifestServer serves the manifest built from the server URL at
// /sdk-versions.json, the mock SDK for every /download/ path, and any extra
// archives by base name. It counts manifest requests.
func manifestServer(t *testing.T, extra map[string][]byte, build func(base string, sha string) dist.Manifest) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	sdkData, sha := testutil.MockSDKZip()
	var manifestRequests atomic.Int32
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	manifest := build(server.URL, sha)
	mux.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		if data, ok := extra[filepath.Base(r.URL.Path)]; ok {
			_, _ = w.Write(data)
			return
		}
		_, _ = w.Write(sdkData)
	})
	mux.HandleFunc("/sdk-versions.json", func(w http.ResponseWriter, _ *http.Request) {
		manifestRequests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		assert.NoError(t, json.NewEncoder(w).Encode(manifest))
	})
	return server, &manifestRequests
}

func download(base, sha, name string) dist.DownloadInfo {
	return dist.DownloadInfo{Name: name, SHA256: sha, URL: base + "/download/" + name}
}

// buildStdxZip writes a minimal stdx zip whose single top-level directory is
// stripped on install, leaving dynamic/ and static/ at the StdxDir root.
func buildStdxZip(t *testing.T, topLevel string) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range map[string]string{
		topLevel + "/dynamic/libfoo.so": "dynamic",
		topLevel + "/static/libfoo.a":   "static",
	} {
		f, err := w.Create(name)
		require.NoError(t, err)
		_, err = f.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())
	return buf.Bytes()
}

func TestInstall_TargetStdxLandsUnderTargetToolchain(t *testing.T) {
	const version = "2.0.0"
	hostKey, err := sdktarget.CurrentHostTuple("")
	require.NoError(t, err)
	ohosKey, err := sdktarget.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	// The target tuple's environment "ohos" maps to the stdx platform token
	// "ohos-aarch64", so the asset name is fixed regardless of the host arch.
	stdxAsset := "cangjie-stdx-ohos-aarch64-" + version + ".1.zip"
	server, _ := manifestServer(t, map[string][]byte{stdxAsset: buildStdxZip(t, "cangjie-stdx-ohos-aarch64-"+version+".1")},
		func(base, sha string) dist.Manifest {
			var manifest dist.Manifest
			manifest.Channels.LTS = dist.ChannelInfo{
				Latest:   "1.0.5",
				Versions: map[string]map[string]dist.DownloadInfo{"1.0.5": {hostKey: download(base, sha, "cangjie-sdk-1.0.5.zip")}},
			}
			manifest.Channels.STS = dist.ChannelInfo{
				Latest: version,
				Versions: map[string]map[string]dist.DownloadInfo{version: {
					hostKey: download(base, sha, "cangjie-sdk-"+version+".zip"),
					ohosKey: download(base, sha, "cangjie-sdk-"+ohosKey+"-"+version+".zip"),
				}},
				// The manifest carries the verbatim download link for the stdx platform.
				Components: map[string]dist.ComponentSet{version: {Stdx: map[string]dist.ComponentInfo{
					"ohos-aarch64": {Name: stdxAsset, URL: base + "/download/" + stdxAsset},
				}}},
			}
			return manifest
		})
	installHome(t, server.URL+"/sdk-versions.json")

	require.NoError(t, install(t, lifecycle.InstallRequest{Toolchain: "sts", Targets: []string{"ohos"}, Components: []string{"stdx"}}))

	targetName := "sts-" + version + "-" + ohosKey
	roots, err := component.RootsFor(targetName)
	require.NoError(t, err)
	assert.True(t, component.IsInstalled(roots.TcDir, component.Stdx), "stdx manifest should exist under the target toolchain dir")
	hostRoots, err := component.RootsFor("sts-" + version)
	require.NoError(t, err)
	assert.False(t, component.IsInstalled(hostRoots.TcDir, component.Stdx), "stdx must NOT be installed against the host toolchain when cross-compiling")
	assert.FileExists(t, filepath.Join(roots.StdxDir, "dynamic", "libfoo.so"))
	assert.FileExists(t, filepath.Join(roots.StdxDir, "static", "libfoo.a"))
}

func TestInstall_PinsTargetToHostVersion(t *testing.T) {
	hostKey, err := sdktarget.CurrentHostTuple("")
	require.NoError(t, err)
	ohosKey, err := sdktarget.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	// STS latest is 2.1.0 (host only); the ohos target build lags at 2.0.0.
	// Resolving host and target independently would install sts-2.1.0 and
	// sts-2.0.0-<ohos>, a version skew that later breaks `envsetup --target`.
	// The target is pinned to the host's resolved version (2.1.0), where no
	// ohos build exists, so the install must fail clearly.
	server, _ := manifestServer(t, nil, func(base, sha string) dist.Manifest {
		var manifest dist.Manifest
		manifest.Channels.LTS = dist.ChannelInfo{
			Latest:   "1.0.5",
			Versions: map[string]map[string]dist.DownloadInfo{"1.0.5": {hostKey: download(base, sha, "lts-1.0.5.zip")}},
		}
		manifest.Channels.STS = dist.ChannelInfo{
			Latest: "2.1.0",
			Versions: map[string]map[string]dist.DownloadInfo{
				"2.0.0": {hostKey: download(base, sha, "sts-2.0.0.zip"), ohosKey: download(base, sha, "sts-2.0.0-ohos.zip")},
				"2.1.0": {hostKey: download(base, sha, "sts-2.1.0.zip")},
			},
		}
		return manifest
	})
	installHome(t, server.URL+"/sdk-versions.json")

	err = install(t, lifecycle.InstallRequest{Toolchain: "sts", Targets: []string{"ohos"}})
	require.Error(t, err, "install must fail when the target SDK lacks the host's resolved version")
	installed := installedNames(t)
	// The host must have been installed first, proving the target-pin code ran.
	assert.Contains(t, installed, "sts-2.1.0", "host toolchain should have installed before the target failure")
	assert.NotContains(t, installed, "sts-2.0.0-"+ohosKey, "must not install a version-skewed target SDK")
}

func TestInstall_FetchesManifestOnce(t *testing.T) {
	hostKey, err := sdktarget.CurrentHostTuple("")
	require.NoError(t, err)
	ohosKey, err := sdktarget.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	androidKey, err := sdktarget.CurrentTargetTuple("", "android")
	require.NoError(t, err)
	server, manifestRequests := manifestServer(t, nil, func(base, sha string) dist.Manifest {
		var manifest dist.Manifest
		manifest.Channels.LTS = dist.ChannelInfo{
			Latest:   "1.0.5",
			Versions: map[string]map[string]dist.DownloadInfo{"1.0.5": {hostKey: download(base, sha, "cangjie-sdk-1.0.5.zip")}},
		}
		manifest.Channels.STS = dist.ChannelInfo{
			Latest: "2.0.0",
			Versions: map[string]map[string]dist.DownloadInfo{"2.0.0": {
				hostKey:    download(base, sha, "cangjie-sdk-2.0.0.zip"),
				ohosKey:    download(base, sha, "cangjie-sdk-"+ohosKey+"-2.0.0.zip"),
				androidKey: download(base, sha, "cangjie-sdk-"+androidKey+"-2.0.0.zip"),
			}},
		}
		return manifest
	})
	installHome(t, server.URL+"/sdk-versions.json")

	require.NoError(t, install(t, lifecycle.InstallRequest{Toolchain: "sts", Targets: []string{"ohos", "android"}}))
	assert.Equal(t, int32(1), manifestRequests.Load())
}

// Tests for Distribution.Resolve.

func TestResolveUsesLatestVersionAvailableForTarget(t *testing.T) {
	installHome(t, testutil.ManifestOnlyServer(t, testutil.ManifestWithPlatformGap()).URL+"/sdk-versions.json")
	d, err := lifecycle.OpenDistribution(lifecycle.Options{})
	require.NoError(t, err)

	resolved, err := d.Resolve(context.Background(), toolchain.ToolchainName{Channel: toolchain.LTS}, "linux-x64")
	require.NoError(t, err)
	assert.Equal(t, "lts-1.5.0", resolved.Name)
	assert.Equal(t, "cangjie-sdk-linux-x64-1.5.0.tar.gz", resolved.ArchiveName)
}

func TestResolveTargetUsesManifestNightlyVersion(t *testing.T) {
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

	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv(config.EnvDistServer, "")
	settings := config.DefaultSettings()
	settings.DefaultHost = "linux-amd64"
	settings.ManifestURL = server.URL
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))
	d, err := lifecycle.OpenDistribution(lifecycle.Options{})
	require.NoError(t, err)
	assert.Equal(t, "linux-x64", d.HostTuple)

	host, err := d.Resolve(context.Background(), toolchain.ToolchainName{Channel: toolchain.Nightly}, "")
	require.NoError(t, err)
	assert.Equal(t, "nightly-"+version, host.Name)

	tuple, err := d.TargetTuple("ohos")
	require.NoError(t, err)
	target, err := d.Resolve(context.Background(), toolchain.ToolchainName{Channel: toolchain.Nightly, Version: version}, tuple)
	require.NoError(t, err)
	assert.Equal(t, "nightly-"+version+"-linux-x64-ohos", target.Name)
	assert.True(t, strings.HasSuffix(target.URL, "/ohos.tar.gz"), target.URL)

	_, err = d.TargetTuple("linux-x64")
	require.Error(t, err, "a full tuple is not an environment")
}

func TestOpenDistributionRejectsInvalidSource(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv(config.EnvDistServer, "")
	settings := config.DefaultSettings()
	settings.DistServer = "ftp://example.invalid/cjv"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))
	_, err := lifecycle.OpenDistribution(lifecycle.Options{})
	require.Error(t, err)
	_, statErr := os.Stat(filepath.Join(home, "toolchains"))
	assert.True(t, os.IsNotExist(statErr), "opening the distribution creates nothing")
}
