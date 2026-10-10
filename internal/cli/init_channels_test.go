package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func initChannelServer(t *testing.T, failStable bool) (*httptest.Server, *atomic.Int32) {
	t.Helper()
	archive, hash := testutil.MockTarGz(t, map[string]string{"cangjie/bin/cjc": "stub", "cangjie/tools/bin/cjpm": "stub"})
	info := dist.DownloadInfo{Name: "sdk.tar.gz", URL: "sdk.tar.gz", SHA256: hash}
	channel := func(version string, tuples ...string) dist.ChannelInfo {
		platforms := map[string]dist.DownloadInfo{}
		for _, tuple := range tuples {
			platforms[tuple] = info
		}
		return dist.ChannelInfo{Latest: version, Versions: map[string]map[string]dist.DownloadInfo{version: platforms}}
	}
	var manifest dist.Manifest
	manifest.Channels.LTS = channel("1.0.5", "linux-x64", "linux-x64-ohos")
	manifest.Channels.LTS.Versions["1.0.4"] = map[string]dist.DownloadInfo{"darwin-x64": info}
	manifest.Channels.STS = channel("1.2.0", "linux-x64", "darwin-arm64")
	nightly := channel("1.3.0-alpha.20261010001050", "linux-x64", "ohos-arm64")
	downloads := new(atomic.Int32)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/versions.json":
			if failStable {
				http.Error(w, "unavailable", http.StatusForbidden)
				return
			}
			_ = json.NewEncoder(w).Encode(manifest)
		case "/nightly.json":
			_ = json.NewEncoder(w).Encode(nightly)
		case "/sdk.tar.gz":
			downloads.Add(1)
			_, _ = w.Write(archive)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server, downloads
}

func TestInitChannelsUseNativeHostManifestAvailability(t *testing.T) {
	server, downloads := initChannelServer(t, false)
	settings := config.DefaultSettings()
	t.Setenv(config.EnvDistServer, "")
	settings.DistServer = server.URL
	for _, test := range []struct {
		tuple     string
		want      []string
		preferred string
	}{
		{"linux-x64", []string{"lts", "sts", "nightly"}, "lts"},
		{"ohos-arm64", []string{"nightly"}, "nightly"},
		{"darwin-arm64", []string{"sts"}, "sts"},
		{"darwin-x64", []string{"lts"}, "lts"},
		{"ohos-x64", nil, "none"},
	} {
		t.Run(test.tuple, func(t *testing.T) {
			source, err := dist.NewSource(&settings)
			require.NoError(t, err)
			channels, err := initAvailableChannels(t.Context(), source, test.tuple)
			require.NoError(t, err)
			assert.Equal(t, test.want, channels)
			assert.Equal(t, test.preferred, initPreferredToolchain(channels))
			opts := initCustomizeOptions{toolchain: "auto", channels: channels}
			options := initToolchainOptions(&opts)
			values := make([]string, 0, len(options))
			for _, option := range options {
				values = append(values, option.Value)
			}
			assert.Equal(t, append(append([]string{}, test.want...), "none"), values)
			assert.Equal(t, test.preferred, opts.toolchain)
		})
	}
	assert.Zero(t, downloads.Load(), "discovery only reads metadata")
}

func TestRunInitAutoWithoutAvailableChannel(t *testing.T) {
	for _, canceled := range []bool{false, true} {
		t.Run(map[bool]string{false: "no-compatible-build", true: "canceled"}[canceled], func(t *testing.T) {
			home := t.TempDir()
			config.IsolateForTest(t, home)
			t.Setenv(config.EnvDistServer, "")
			t.Setenv(config.EnvNoPathSetup, "1")
			server, downloads := initChannelServer(t, false)
			settings := config.DefaultSettings()
			settings.DistServer = server.URL
			settings.DefaultHost = "openharmony-amd64"
			path, err := config.SettingsPath()
			require.NoError(t, err)
			require.NoError(t, config.SaveSettings(&settings, path))
			config.ResetDefaultSettingsFileCache()
			t.Cleanup(config.ResetDefaultSettingsFileCache)
			app := newApplication("dev", "")
			app.initYes = true
			app.initNoModifyPath = true
			var out, warnings bytes.Buffer
			cmd := &cobra.Command{}
			cmd.SetOut(&out)
			cmd.SetErr(&warnings)
			if canceled {
				ctx, cancel := context.WithCancel(t.Context())
				cancel()
				cmd.SetContext(ctx)
			}
			err = app.runInit(cmd, nil)
			managed := filepath.Join(home, "bin", sdktools.CjvBinaryName())
			if canceled {
				require.ErrorIs(t, err, context.Canceled)
				assert.NoFileExists(t, managed)
			} else {
				require.Error(t, err)
				assert.NoFileExists(t, managed)
				assert.Contains(t, err.Error(), "ohos-x64")
				assert.Contains(t, err.Error(), "--default-toolchain none")
			}
			assert.Zero(t, downloads.Load())
		})
	}
}

func TestRunInitAutoInstallsNightlyForHarmonyHost(t *testing.T) {
	for _, failStable := range []bool{false, true} {
		t.Run(map[bool]string{false: "nightly-only-host", true: "stable-manifest-failed"}[failStable], func(t *testing.T) {
			home := t.TempDir()
			config.IsolateForTest(t, home)
			t.Setenv(config.EnvDistServer, "")
			t.Setenv(config.EnvNoPathSetup, "1")
			server, downloads := initChannelServer(t, failStable)
			settings := config.DefaultSettings()
			settings.DistServer = server.URL
			settings.DefaultHost = "openharmony-arm64"
			path, err := config.SettingsPath()
			require.NoError(t, err)
			require.NoError(t, config.SaveSettings(&settings, path))
			config.ResetDefaultSettingsFileCache()
			t.Cleanup(config.ResetDefaultSettingsFileCache)
			app := newApplication("dev", "")
			app.initYes = true
			app.initNoModifyPath = true
			var out, warnings bytes.Buffer
			cmd := &cobra.Command{}
			cmd.SetContext(t.Context())
			cmd.SetOut(&out)
			cmd.SetErr(&warnings)
			require.NoError(t, app.runInit(cmd, nil))
			assert.Equal(t, int32(1), downloads.Load())
			assert.DirExists(t, filepath.Join(home, "toolchains", "nightly"))
			assert.FileExists(t, filepath.Join(home, "bin", sdktools.CjvBinaryName()))
			if failStable {
				assert.Contains(t, warnings.String(), "HTTP 403")
			} else {
				assert.Empty(t, warnings.String())
			}
		})
	}
}
