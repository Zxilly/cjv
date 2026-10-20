package lifecycle

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/target"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/require"
)

func TestChannelGroupRetainsEverySDKWhenTargetIsUnavailable(t *testing.T) {
	for _, operation := range []string{"install", "update", "all"} {
		t.Run(operation, func(t *testing.T) {
			home := updateHome(t)
			selectRelease(t, "1.0.0", "ohos")
			require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts", Targets: []string{"ohos"}}, quietLifecycleOptions()))
			tuple, err := sdktarget.CurrentTargetTuple("", "ohos")
			require.NoError(t, err)
			selectRelease(t, "2.0.0")
			switch operation {
			case "install":
				err = Install(t.Context(), InstallRequest{Toolchain: "sts"}, quietLifecycleOptions())
			case "update":
				_, err = UpdateInstalled(t.Context(), parse(t, "sts"), quietLifecycleOptions())
			case "all":
				_, err = UpdateAll(t.Context(), quietLifecycleOptions())
			}
			require.Error(t, err)
			require.Equal(t, "sts-1.0.0", readRelease(t, home, "sts").Release)
			require.Equal(t, "sts-1.0.0-"+tuple, readRelease(t, home, "sts-"+tuple).Release)
		})
	}
}

func TestNoUpdateWithTargetsIsOffline(t *testing.T) {
	home := updateHome(t)
	selectRelease(t, "1.0.0", "ohos")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts", Targets: []string{"ohos"}}, Options{}))
	selectRelease(t, "2.0.0")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts", NoUpdate: true}, Options{}))
	require.Equal(t, "sts-1.0.0", readRelease(t, home, "sts").Release)
}

func TestExplicitCurrentHostHasOneTargetOwner(t *testing.T) {
	updateHome(t)
	selectRelease(t, "1.0.0", "ohos")
	host, err := target.CurrentHostTuple("")
	require.NoError(t, err)
	for _, input := range []string{"sts", "sts-" + host} {
		require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: input, Targets: []string{"ohos"}}, Options{}))
	}
	plain, err := toolchain.FindInstalled(parse(t, "sts"))
	require.NoError(t, err)
	explicit, err := toolchain.FindInstalled(parse(t, "sts-"+host))
	require.NoError(t, err)
	require.Equal(t, plain, explicit)
	installed, err := toolchain.ListInstalled()
	require.NoError(t, err)
	require.Len(t, installed, 2)
}

func TestChannelGroupRecoversRealProcessInterruption(t *testing.T) {
	if os.Getenv("CJV_TEST_GROUP_CRASH") == "1" {
		configtest.Isolate(t, os.Getenv(config.EnvHome))
		afterFinalizeHook = func() error { os.Exit(73); return nil }
		_ = Install(t.Context(), InstallRequest{Toolchain: "sts"}, Options{})
		os.Exit(74)
	}
	home := updateHome(t)
	selectRelease(t, "1.0.0", "ohos")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts", Targets: []string{"ohos"}}, Options{}))
	tuple, err := target.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	selectRelease(t, "2.0.0", "ohos")
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestChannelGroupRecoversRealProcessInterruption$")
	cmd.Env = append(os.Environ(), "CJV_TEST_GROUP_CRASH=1")
	output, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr, "%s", output)
	require.Equal(t, 73, exitErr.ExitCode(), "%s", output)
	require.Equal(t, "sts-2.0.0", readRelease(t, home, "sts").Release)
	require.Equal(t, "sts-2.0.0-"+tuple, readRelease(t, home, "sts-"+tuple).Release)
	require.NoError(t, toolchain.RecoverHome())
	require.Equal(t, "sts-1.0.0", readRelease(t, home, "sts").Release)
	require.Equal(t, "sts-1.0.0-"+tuple, readRelease(t, home, "sts-"+tuple).Release)
}

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
