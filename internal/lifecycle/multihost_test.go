package lifecycle

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/stretchr/testify/require"
)

func TestDifferentHostsOwnIndependentTargetGroups(t *testing.T) {
	home := updateHome(t)
	host, err := target.CurrentHostTuple("")
	require.NoError(t, err)
	other := "linux-x64"
	if host == other {
		other = "win32-x64"
	}
	sdk, checksum := testutil.MockTarGz(t, map[string]string{"cangjie/bin/cjc": "compiler", "cangjie/bin/cjc.exe": "compiler"})
	var latest atomic.Value
	latest.Store("1.0.0")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sdk.tar.gz" {
			_, _ = w.Write(sdk)
			return
		}
		version := latest.Load().(string)
		platforms := map[string]dist.DownloadInfo{}
		for _, tuple := range []string{host, other, host + "-ohos", other + "-ohos"} {
			platforms[tuple] = dist.DownloadInfo{Name: "sdk.tar.gz", URL: "http://" + r.Host + "/sdk.tar.gz", SHA256: checksum}
		}
		info := dist.ChannelInfo{Latest: version, Versions: map[string]map[string]dist.DownloadInfo{version: platforms}}
		var manifest dist.Manifest
		manifest.Channels.STS = info
		manifest.Channels.LTS = info
		_ = json.NewEncoder(w).Encode(manifest)
	}))
	defer server.Close()
	sf, err := config.DefaultSettingsFile()
	require.NoError(t, err)
	url := server.URL + "/versions.json"
	_, err = sf.Update(config.SettingsUpdate{ManifestURL: &url})
	require.NoError(t, err)
	for _, input := range []string{"sts", "sts-" + other} {
		require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: input, Targets: []string{"ohos"}}, Options{}))
	}
	latest.Store("2.0.0")
	_, err = UpdateAll(t.Context(), Options{})
	require.NoError(t, err)
	require.Equal(t, "sts-2.0.0", readRelease(t, home, "sts").Release)
	require.Equal(t, "sts-2.0.0-"+other, readRelease(t, home, "sts-"+other).Release)
	require.Equal(t, "sts-2.0.0-"+host+"-ohos", readRelease(t, home, "sts-"+host+"-ohos").Release)
	require.NoError(t, RemoveToolchain("sts-"+other))
	require.DirExists(t, filepath.Join(home, "toolchains", "sts-"+host+"-ohos"))
	require.NoDirExists(t, filepath.Join(home, "toolchains", "sts-"+other+"-ohos"))
}

func TestExplicitHostAliasRetainsArtifactTupleAfterDefaultHostChanges(t *testing.T) {
	updateHome(t)
	selectRelease(t, "1.0.0")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, Options{}))
	d, err := OpenDistribution(Options{})
	require.NoError(t, err)
	original := d.HostTuple
	if original == "linux-x64" {
		d.HostTuple = "win32-x64"
	} else {
		d.HostTuple = "linux-x64"
	}
	resolved, err := d.Resolve(t.Context(), parse(t, "sts-"+original), "")
	require.NoError(t, err)
	require.Equal(t, original, resolved.Tuple)
	require.Equal(t, "sts-1.0.0", resolved.Name)
}
