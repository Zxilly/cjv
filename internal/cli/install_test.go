package cli

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
	componentlib "github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createMockSDK builds a zip with platform-correct binary names.
func createMockSDK() ([]byte, string) {
	return createMockSDKWithEnvSetup(true)
}

func createMockSDKWithEnvSetup(includeEnvSetup bool) ([]byte, string) {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)

	writeEntry := func(name, content string) {
		f, err := w.Create(name)
		if err != nil {
			panic(fmt.Sprintf("zip create %s: %v", name, err))
		}
		if _, err := f.Write([]byte(content)); err != nil {
			panic(fmt.Sprintf("zip write %s: %v", name, err))
		}
	}

	// Add all proxy tools at their expected relative paths
	for _, tool := range proxy.AllProxyTools() {
		relPath := proxy.ToolRelativePath(tool)
		name := "cangjie/" + relPath
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		writeEntry(name, "stub-"+tool)
	}
	if includeEnvSetup {
		writeEntry("cangjie/envsetup.sh", "export CANGJIE_HOME=\"$PWD\"")
		writeEntry("cangjie/envsetup.ps1", "$env:CANGJIE_HOME = $PWD.Path")
	}

	w.Close()
	hash := sha256.Sum256(buf.Bytes())
	return buf.Bytes(), hex.EncodeToString(hash[:])
}

// Creates a mock distribution server with a valid manifest.
func validMockServer(t *testing.T) *httptest.Server {
	t.Helper()
	sdkData, sha := createMockSDK()
	pk, err := dist.CurrentHostTuple("")
	require.NoError(t, err)

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	var manifest dist.Manifest
	manifest.Channels.LTS = dist.ChannelInfo{
		Latest: "1.0.5",
		Versions: map[string]map[string]dist.DownloadInfo{
			"1.0.5": {
				pk: {
					Name:   "cangjie-sdk-1.0.5.zip",
					SHA256: sha,
					URL:    server.URL + "/download/cangjie-sdk-1.0.5.zip",
				},
			},
		},
	}
	manifest.Channels.STS = dist.ChannelInfo{
		Latest: "2.0.0",
		Versions: map[string]map[string]dist.DownloadInfo{
			"2.0.0": {
				pk: {
					Name:   "cangjie-sdk-2.0.0.zip",
					SHA256: sha,
					URL:    server.URL + "/download/cangjie-sdk-2.0.0.zip",
				},
			},
		},
	}

	mux.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(sdkData)
	})
	mux.HandleFunc("/sdk-versions.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(manifest); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	return server
}

func mockServerWithSDK(t *testing.T, sdkData []byte, sha string) *httptest.Server {
	t.Helper()
	pk, err := dist.CurrentHostTuple("")
	require.NoError(t, err)

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	var manifest dist.Manifest
	manifest.Channels.LTS = dist.ChannelInfo{
		Latest: "1.0.5",
		Versions: map[string]map[string]dist.DownloadInfo{
			"1.0.5": {
				pk: {
					Name:   "cangjie-sdk-1.0.5.zip",
					SHA256: sha,
					URL:    server.URL + "/download/cangjie-sdk-1.0.5.zip",
				},
			},
		},
	}

	mux.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(sdkData)
	})
	mux.HandleFunc("/sdk-versions.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(manifest); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	return server
}

func mockServerWithTargetSDKs(t *testing.T, channel toolchain.Channel, version string, targets ...string) *httptest.Server {
	t.Helper()
	sdkData, sha := createMockSDK()
	hostKey, err := dist.CurrentHostTuple("")
	require.NoError(t, err)

	platforms := map[string]dist.DownloadInfo{
		hostKey: {
			Name:   "cangjie-sdk-" + version + ".zip",
			SHA256: sha,
			URL:    "",
		},
	}
	for _, target := range targets {
		key, err := dist.CurrentTargetTuple("", target)
		require.NoError(t, err)
		platforms[key] = dist.DownloadInfo{
			Name:   "cangjie-sdk-" + key + "-" + version + ".zip",
			SHA256: sha,
			URL:    "",
		}
	}

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	for key, info := range platforms {
		info.URL = server.URL + "/download/" + info.Name
		platforms[key] = info
	}

	var manifest dist.Manifest
	if channel == toolchain.LTS {
		manifest.Channels.LTS = dist.ChannelInfo{
			Latest:   version,
			Versions: map[string]map[string]dist.DownloadInfo{version: platforms},
		}
		manifest.Channels.STS = dist.ChannelInfo{
			Latest: "2.0.0",
			Versions: map[string]map[string]dist.DownloadInfo{
				"2.0.0": {
					hostKey: {Name: "cangjie-sdk-2.0.0.zip", SHA256: sha, URL: server.URL + "/download/cangjie-sdk-2.0.0.zip"},
				},
			},
		}
	} else {
		manifest.Channels.LTS = dist.ChannelInfo{
			Latest: "1.0.5",
			Versions: map[string]map[string]dist.DownloadInfo{
				"1.0.5": {
					hostKey: {Name: "cangjie-sdk-1.0.5.zip", SHA256: sha, URL: server.URL + "/download/cangjie-sdk-1.0.5.zip"},
				},
			},
		}
		manifest.Channels.STS = dist.ChannelInfo{
			Latest:   version,
			Versions: map[string]map[string]dist.DownloadInfo{version: platforms},
		}
	}

	mux.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(sdkData)
	})
	mux.HandleFunc("/sdk-versions.json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(manifest); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	return server
}

// Integration-style tests for the install flow.
// These test the full pipeline: resolve -> download -> extract -> validate -> swap.

func TestInstallToolchainWithOptions_InstallsLTS(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)

	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	err := InstallToolchainWithOptions(context.Background(), "lts", false)
	require.NoError(t, err)

	installed, err := toolchain.ListInstalled()
	require.NoError(t, err)
	assert.NotEmpty(t, installed, "should have at least one installed toolchain")
}

func TestInstallToolchainWithTargets_InstallsHostAndTargets(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	server := mockServerWithTargetSDKs(t, toolchain.STS, "2.0.0", "ohos", "android")
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	err := InstallToolchainWithTargets(context.Background(), "sts", []string{"ohos", "android"}, false)
	require.NoError(t, err)

	hostKey, err := dist.CurrentHostTuple("")
	require.NoError(t, err)
	ohosKey, err := dist.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	androidKey, err := dist.CurrentTargetTuple("", "android")
	require.NoError(t, err)

	installed, err := toolchain.ListInstalled()
	require.NoError(t, err)
	assert.Contains(t, installed, "sts-2.0.0")
	assert.NotContains(t, installed, "sts-2.0.0-"+hostKey)
	assert.Contains(t, installed, "sts-2.0.0-"+ohosKey)
	assert.Contains(t, installed, "sts-2.0.0-"+androidKey)

	reloaded, err := config.LoadSettings(filepath.Join(home, ".cjv", "settings.toml"))
	require.NoError(t, err)
	assert.Equal(t, "sts-2.0.0", reloaded.DefaultToolchain)
}

func TestInstallToolchainWithTargets_FetchesManifestOnce(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	sdkData, sha := createMockSDK()
	hostKey, err := dist.CurrentHostTuple("")
	require.NoError(t, err)
	ohosKey, err := dist.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	androidKey, err := dist.CurrentTargetTuple("", "android")
	require.NoError(t, err)

	var manifestRequests atomic.Int32
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	platforms := map[string]dist.DownloadInfo{
		hostKey: {
			Name:   "cangjie-sdk-2.0.0.zip",
			SHA256: sha,
			URL:    server.URL + "/download/cangjie-sdk-2.0.0.zip",
		},
		ohosKey: {
			Name:   "cangjie-sdk-" + ohosKey + "-2.0.0.zip",
			SHA256: sha,
			URL:    server.URL + "/download/cangjie-sdk-" + ohosKey + "-2.0.0.zip",
		},
		androidKey: {
			Name:   "cangjie-sdk-" + androidKey + "-2.0.0.zip",
			SHA256: sha,
			URL:    server.URL + "/download/cangjie-sdk-" + androidKey + "-2.0.0.zip",
		},
	}
	var manifest dist.Manifest
	manifest.Channels.LTS = dist.ChannelInfo{
		Latest: "1.0.5",
		Versions: map[string]map[string]dist.DownloadInfo{
			"1.0.5": {
				hostKey: {
					Name:   "cangjie-sdk-1.0.5.zip",
					SHA256: sha,
					URL:    server.URL + "/download/cangjie-sdk-1.0.5.zip",
				},
			},
		},
	}
	manifest.Channels.STS = dist.ChannelInfo{
		Latest:   "2.0.0",
		Versions: map[string]map[string]dist.DownloadInfo{"2.0.0": platforms},
	}

	mux.HandleFunc("/download/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = w.Write(sdkData)
	})
	mux.HandleFunc("/sdk-versions.json", func(w http.ResponseWriter, r *http.Request) {
		manifestRequests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(manifest); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	require.NoError(t, InstallToolchainWithTargets(context.Background(), "sts", []string{"ohos", "android"}, false))
	assert.Equal(t, int32(1), manifestRequests.Load())
}

func TestInstallToolchainWithTargets_BareVersionResolvesChannel(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	server := mockServerWithTargetSDKs(t, toolchain.STS, "2.0.0", "ohos")
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	require.NoError(t, InstallToolchainWithTargets(context.Background(), "2.0.0", []string{"ohos"}, false))

	ohosKey, err := dist.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	installed, err := toolchain.ListInstalled()
	require.NoError(t, err)
	assert.Contains(t, installed, "sts-2.0.0")
	assert.Contains(t, installed, "sts-2.0.0-"+ohosKey)
}

func TestInstallToolchainWithTargets_ExplicitVariantDoesNotSetDefault(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	server := mockServerWithTargetSDKs(t, toolchain.STS, "2.0.0", "ohos")
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	ohosKey, err := dist.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	require.NoError(t, InstallToolchainWithTargets(context.Background(), "sts-2.0.0-"+ohosKey, nil, false))

	installed, err := toolchain.ListInstalled()
	require.NoError(t, err)
	assert.Contains(t, installed, "sts-2.0.0-"+ohosKey)
	assert.NotContains(t, installed, "sts-2.0.0")

	reloaded, err := config.LoadSettings(filepath.Join(home, ".cjv", "settings.toml"))
	require.NoError(t, err)
	assert.Empty(t, reloaded.DefaultToolchain)
}

func TestInstallToolchainWithTargets_RejectsVariantPlusTargets(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	settings := config.DefaultSettings()
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	ohosKey, err := dist.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	err = InstallToolchainWithTargets(context.Background(), "sts-2.0.0-"+ohosKey, []string{"android"}, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot combine target variant")
}

func TestInstallToolchainWithOptions_AlreadyInstalled(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)

	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	require.NoError(t, InstallToolchainWithOptions(context.Background(), "lts", false))

	// Second install without force — prints "already installed" and returns nil
	err := InstallToolchainWithOptions(context.Background(), "lts", false)
	assert.NoError(t, err, "already-installed is an informational no-op, not an error")
}

func TestInstallToolchainWithOptions_ForceReinstall(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)

	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	require.NoError(t, InstallToolchainWithOptions(context.Background(), "lts", false))

	err := InstallToolchainWithOptions(context.Background(), "lts", true)
	assert.NoError(t, err, "force install should succeed even when already installed")
}

func TestInstallToolchainWithOptions_SetsDefault(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)

	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	require.NoError(t, InstallToolchainWithOptions(context.Background(), "lts", false))

	reloaded, err := config.LoadSettings(filepath.Join(home, ".cjv", "settings.toml"))
	require.NoError(t, err)
	assert.NotEmpty(t, reloaded.DefaultToolchain,
		"first install should set the default toolchain")
}

func TestInstallToolchainWithOptions_BootstrapsManagedBinary(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)

	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	binDir, err := config.BinDir()
	require.NoError(t, err)
	managedBinary := filepath.Join(binDir, proxy.CjvBinaryName())
	_, err = os.Stat(managedBinary)
	require.Error(t, err)

	err = InstallToolchainWithOptions(context.Background(), "lts", false)
	require.NoError(t, err)

	assert.FileExists(t, managedBinary, "first install should bootstrap the managed cjv binary")
}

func TestInstallToolchainWithOptions_FailsWhenEnvSetupMissing(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	sdkData, sha := createMockSDKWithEnvSetup(false)
	server := mockServerWithSDK(t, sdkData, sha)

	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	err := InstallToolchainWithOptions(context.Background(), "lts", false)
	require.Error(t, err)
}

// Tests for fetchManifest -- fetches and parses the SDK version manifest.

func TestFetchManifest_ValidManifest(t *testing.T) {
	server := validMockServer(t)

	manifest, err := fetchManifest(context.Background(), server.URL+"/sdk-versions.json")
	require.NoError(t, err)
	assert.NotNil(t, manifest)
}

func TestFetchManifest_InvalidURL(t *testing.T) {
	_, err := fetchManifest(context.Background(), "http://localhost:1/nonexistent")
	assert.Error(t, err)
}

func TestFetchManifest_HTTPError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	_, err := fetchManifest(context.Background(), server.URL+"/sdk-versions.json")
	assert.Error(t, err, "should fail on HTTP 404")
}

// Tests for install with a bare version number (e.g., "1.0.5" instead
// of "lts-1.0.5"). resolveAndLocate must discover the channel from
// the manifest.

func TestInstallToolchainWithOptions_BareVersion(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	// Install by bare version — system should discover channel from manifest
	require.NoError(t, InstallToolchainWithOptions(context.Background(), "1.0.5", false))

	installed, _ := toolchain.ListInstalled()
	assert.Contains(t, installed, "lts-1.0.5",
		"bare version should resolve to correct channel")
}

func TestInstallToolchainWithOptions_BareVersionNotFound(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	err := InstallToolchainWithOptions(context.Background(), "99.99.99", false)
	assert.Error(t, err, "non-existent version should fail")
}

func TestInstallToolchainWithOptions_InvalidName(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)

	err := InstallToolchainWithOptions(context.Background(), "+invalid", false)
	assert.Error(t, err, "invalid name starting with + should fail")
}

// Tests install flow with STS channel — exercises different code paths
// in resolveAndLocate (different channel lookup).

func TestInstallToolchainWithOptions_STS(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	require.NoError(t, InstallToolchainWithOptions(context.Background(), "sts", false))

	installed, _ := toolchain.ListInstalled()
	assert.Contains(t, installed, "sts-2.0.0")
}

func TestInstallToolchainWithOptions_SpecificVersion(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	require.NoError(t, InstallToolchainWithOptions(context.Background(), "lts-1.0.5", false))

	installed, _ := toolchain.ListInstalled()
	assert.Contains(t, installed, "lts-1.0.5")
}

// Full lifecycle: install -> check -> update -> uninstall

func TestFullLifecycle(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	t.Chdir(cwd)

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	// Install
	require.NoError(t, InstallToolchainWithOptions(context.Background(), "lts", false))
	installed, _ := toolchain.ListInstalled()
	require.NotEmpty(t, installed)

	// Update (already up to date)
	_, updateErr := updateAll(context.Background())
	require.NoError(t, updateErr)

	// Uninstall
	require.NoError(t, runUninstall(nil, []string{installed[0]}))
	remaining, _ := toolchain.ListInstalled()
	assert.Empty(t, remaining)
}

// Test installing bare version "2.0.0" — should discover STS channel.

func TestInstallToolchainWithOptions_BareVersionSTS(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	require.NoError(t, InstallToolchainWithOptions(context.Background(), "2.0.0", false))

	installed, _ := toolchain.ListInstalled()
	assert.Contains(t, installed, "sts-2.0.0")
}

// Test installing both channels in sequence.

func TestInstallBothChannels(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	require.NoError(t, InstallToolchainWithOptions(context.Background(), "lts", false))
	require.NoError(t, InstallToolchainWithOptions(context.Background(), "sts", false))

	installed, _ := toolchain.ListInstalled()
	assert.Len(t, installed, 2)
	assert.Contains(t, installed, "lts-1.0.5")
	assert.Contains(t, installed, "sts-2.0.0")
}

// Tests for validateInstallation -- verifies that an extracted SDK
// contains the essential "cjc" binary.

func TestValidateInstallation_ValidSDK(t *testing.T) {
	dir := t.TempDir()

	// Create the cjc binary at the expected location
	relPath := proxy.ToolRelativePath("cjc")
	cjcPath := filepath.Join(dir, relPath)
	if runtime.GOOS == "windows" {
		cjcPath += ".exe"
	}
	require.NoError(t, os.MkdirAll(filepath.Dir(cjcPath), 0o755))
	require.NoError(t, os.WriteFile(cjcPath, []byte("stub"), 0o755))

	assert.NoError(t, validateInstallation(dir, ""),
		"should pass when cjc binary exists at the expected path")
}

func TestValidateInstallationUsesResolvedTupleBinaryName(t *testing.T) {
	winDir := t.TempDir()
	winCJCPath := filepath.Join(winDir, proxy.ToolRelativePath("cjc")) + ".exe"
	require.NoError(t, os.MkdirAll(filepath.Dir(winCJCPath), 0o755))
	require.NoError(t, os.WriteFile(winCJCPath, []byte("stub"), 0o755))
	assert.NoError(t, validateInstallation(winDir, "win32-x64"))

	linuxDir := t.TempDir()
	linuxCJCPath := filepath.Join(linuxDir, proxy.ToolRelativePath("cjc"))
	require.NoError(t, os.MkdirAll(filepath.Dir(linuxCJCPath), 0o755))
	require.NoError(t, os.WriteFile(linuxCJCPath, []byte("stub"), 0o755))
	assert.NoError(t, validateInstallation(linuxDir, "linux-x64"))
}

func TestValidateInstallation_MissingBinary(t *testing.T) {
	dir := t.TempDir()
	// Empty directory — no cjc binary

	err := validateInstallation(dir, "")
	assert.Error(t, err, "should fail when cjc binary is missing")
}

func TestValidateInstallation_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	err := validateInstallation(dir, "")
	assert.Error(t, err, "should fail on empty directory")
}

// Tests for runInstall -- the cobra handler that parses --force flag.

func TestRunInstall_WithoutForce(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	cmd := &cobra.Command{}
	cmd.Flags().BoolP("force", "f", false, "")
	err := runInstall(cmd, []string{"lts"})
	assert.NoError(t, err)
}

func TestRunInstall_InvalidName(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)

	cmd := &cobra.Command{}
	cmd.Flags().BoolP("force", "f", false, "")
	err := runInstall(cmd, []string{""})
	assert.Error(t, err, "empty name should fail")
}

// Test for InstallToolchainWithOptions — the public API that installs a toolchain.

func TestInstallToolchainWithOptions_Wrapper(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	require.NoError(t, config.EnsureDirs())

	server := validMockServer(t)
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	require.NoError(t, InstallToolchainWithOptions(context.Background(), "lts", false))

	installed, _ := toolchain.ListInstalled()
	assert.NotEmpty(t, installed)
}

func TestResolveNightlyWithSpecificVersionSkipsLatestLookup(t *testing.T) {
	settings := config.DefaultSettings()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	resolved, err := resolveNightlyWithTuple(ctx, toolchain.ToolchainName{
		Channel: toolchain.Nightly,
		Version: "202501010000",
	}, &settings, "linux-x64")

	require.NoError(t, err)
	assert.Equal(t, "nightly-202501010000", resolved.Name)
	assert.Contains(t, resolved.URL, "202501010000")
	assert.Empty(t, resolved.SHA256)
}

func TestInstallToolchainWithExtrasRejectsCustomAndTargetVariantWithTargets(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)

	err := InstallToolchainWithExtras(context.Background(), "local-sdk", nil, nil, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "custom toolchain")

	err = InstallToolchainWithExtras(context.Background(), "lts-1.0.5-linux-x64-ohos", []string{"android"}, nil, false)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot combine")
}

func TestEnsurePathConfiguredCanBeDisabledByEnv(t *testing.T) {
	t.Setenv(config.EnvNoPathSetup, "1")
	ensurePathConfigured()
}

func TestInstallComponentsForToolchainNoComponentsIsNoop(t *testing.T) {
	require.NoError(t, InstallComponentsForToolchain(context.Background(), "lts", nil))
}

func TestInstallComponentsForToolchainRejectsInvalidAndMissingToolchain(t *testing.T) {
	err := InstallComponentsForToolchain(context.Background(), "+bad", []string{"docs"})
	require.Error(t, err)

	home := t.TempDir()
	config.IsolateForTest(t, home)
	config.ResetDefaultSettingsFileCache()
	t.Cleanup(config.ResetDefaultSettingsFileCache)

	err = InstallComponentsForToolchain(context.Background(), "lts-1.0.5", []string{"docs"})
	require.Error(t, err)
}

func TestInstallComponentsListInputValidationAndAlreadyInstalled(t *testing.T) {
	err := installComponentsList(context.Background(), "+bad", []string{"docs"}, false, true)
	require.Error(t, err)

	err = installComponentsList(context.Background(), "local-sdk", []string{"docs"}, false, true)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docs")

	tcName := "lts-1.0.5"
	setupComponentCLITest(t, tcName)
	err = installComponentsList(context.Background(), tcName, []string{"unknown"}, false, true)
	require.Error(t, err)

	oldInstall := componentInstallFunc
	componentInstallFunc = func(ctx context.Context, roots componentlib.Roots, tc toolchain.ToolchainName, name componentlib.Name, tuple, downloadsDir string, force bool) error {
		return &cjverr.ComponentAlreadyInstalledError{Toolchain: tc.String(), Component: string(name)}
	}
	t.Cleanup(func() { componentInstallFunc = oldInstall })

	err = installComponentsList(context.Background(), tcName, []string{"docs"}, false, false)
	require.NoError(t, err)
}

func TestResolveAndLocateDispatchesNightlyAndDefaultToolchainExistsInvalidName(t *testing.T) {
	settings := config.DefaultSettings()
	resolved, err := resolveAndLocateWithTuple(context.Background(), toolchain.ToolchainName{
		Channel: toolchain.Nightly,
		Version: "202501010000",
	}, &settings, newManifestFetcher(""), "linux-x64")

	require.NoError(t, err)
	assert.Equal(t, "nightly-202501010000", resolved.Name)
	assert.False(t, defaultToolchainExists("+bad"))
}
