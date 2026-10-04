package lifecycle

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/require"
)

func TestLegacyTargetComponentInstallKeepsCrossPlatform(t *testing.T) {
	for _, operation := range []string{"migrated-channel-no-update", "fixed-target"} {
		t.Run(operation, func(t *testing.T) {
			home := t.TempDir()
			config.IsolateForTest(t, home)
			t.Setenv(config.EnvDistServer, "")
			host, err := sdktarget.CurrentHostTuple("")
			require.NoError(t, err)
			tuple, err := sdktarget.CurrentTargetTuple("", "ohos")
			require.NoError(t, err)
			platform, err := sdktarget.StdxPlatformForTuple(tuple)
			require.NoError(t, err)
			sdk, sdkSHA := testutil.MockSDKZip()
			stdx, stdxSHA := testutil.MockTarGz(t, map[string]string{
				"stdx/dynamic/target.txt": "ohos stdx",
				"stdx/static/target.txt":  "ohos stdx",
			})
			var sdkRequests, stdxRequests atomic.Int32
			mux := http.NewServeMux()
			server := httptest.NewServer(mux)
			t.Cleanup(server.Close)
			artifact := dist.DownloadInfo{Name: "sdk.zip", URL: server.URL + "/sdk.zip", SHA256: sdkSHA}
			channel := dist.ChannelInfo{
				Latest: "1.0.5",
				Versions: map[string]map[string]dist.DownloadInfo{
					"1.0.5": {host: artifact, tuple: artifact},
				},
				// Deliberately publish only the cross platform's stdx. Falling
				// back to the host platform must fail instead of passing silently.
				Components: map[string]dist.ComponentSet{
					"1.0.5": {Stdx: map[string]dist.ComponentInfo{
						platform: {Name: "stdx.tar.gz", URL: server.URL + "/stdx.tar.gz", SHA256: stdxSHA},
					}},
				},
			}
			manifest := dist.Manifest{}
			manifest.Channels.LTS, manifest.Channels.STS = channel, channel
			mux.HandleFunc("/versions.json", func(w http.ResponseWriter, _ *http.Request) {
				_ = json.NewEncoder(w).Encode(manifest)
			})
			mux.HandleFunc("/sdk.zip", func(w http.ResponseWriter, _ *http.Request) {
				sdkRequests.Add(1)
				_, _ = w.Write(sdk)
			})
			mux.HandleFunc("/stdx.tar.gz", func(w http.ResponseWriter, _ *http.Request) {
				stdxRequests.Add(1)
				_, _ = w.Write(stdx)
			})
			settings := config.DefaultSettings()
			settings.ManifestURL = server.URL + "/versions.json"
			sf, err := config.DefaultSettingsFile()
			require.NoError(t, err)
			require.NoError(t, sf.Save(&settings))

			// Materialize the published legacy layout: an SDK in a concrete
			// version directory, with no cjv release record or layout marker.
			fixed := "lts-1.0.5-" + tuple
			legacyDir := filepath.Join(home, "toolchains", fixed)
			archive := filepath.Join(t.TempDir(), "sdk.zip")
			require.NoError(t, os.WriteFile(archive, sdk, 0o644))
			require.NoError(t, dist.InstallSDK(t.Context(), archive, legacyDir))
			marker := filepath.Join(legacyDir, "preserve.txt")
			require.NoError(t, os.WriteFile(marker, []byte("legacy SDK content"), 0o644))
			require.NoError(t, toolchain.RecoverHome())
			identity := fixed
			noUpdate := false
			if operation == "migrated-channel-no-update" {
				identity, noUpdate = "lts-"+tuple, true
				before, err := toolchain.ReadInstallation(filepath.Join(home, "toolchains", identity))
				require.NoError(t, err)
				require.Empty(t, before.Tuple, "exercise the migrator's release-only record")
			}
			require.NoError(t, Install(t.Context(), InstallRequest{
				Toolchain: identity, NoUpdate: noUpdate, Components: []string{"stdx"},
			}, quietLifecycleOptions()))
			record, err := toolchain.ReadInstallation(filepath.Join(home, "toolchains", identity))
			require.NoError(t, err)
			require.Equal(t, tuple, record.Tuple)
			data, err := os.ReadFile(filepath.Join(home, "stdx", identity, "dynamic", "target.txt"))
			require.NoError(t, err)
			require.Equal(t, "ohos stdx", string(data))
			require.FileExists(t, filepath.Join(home, "toolchains", identity, "preserve.txt"))
			require.FileExists(t, marker, "retain the fixed legacy SDK through migration and component installation")
			require.Zero(t, sdkRequests.Load(), "adding stdx must not redownload an existing SDK")
			require.Positive(t, stdxRequests.Load())
			_, current, err := config.LoadDefaultSettings()
			require.NoError(t, err)
			require.Empty(t, current.DefaultToolchain, "a cross SDK must not become the default host")
		})
	}
}
