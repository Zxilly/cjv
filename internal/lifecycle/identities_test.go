package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/sdktools"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func selectRelease(t *testing.T, version string, targets ...string) {
	t.Helper()
	server := targetSDKServer(t, parse(t, "sts").Channel, version, targets...)
	sf, err := config.DefaultSettingsFile()
	require.NoError(t, err)
	url := server.URL + "/sdk-versions.json"
	_, err = sf.Update(config.SettingsUpdate{ManifestURL: &url})
	require.NoError(t, err)
}

func readChoices(t *testing.T) *config.Settings {
	t.Helper()
	_, settings, err := config.LoadDefaultSettings()
	require.NoError(t, err)
	return settings
}

func readRelease(t *testing.T, home, identity string) toolchain.Installation {
	t.Helper()
	r, err := toolchain.ReadInstallation(filepath.Join(home, "toolchains", identity))
	require.NoError(t, err)
	return r
}

func TestTrackingInstallAndUpdateKeepIndependentDirectories(t *testing.T) {
	for _, command := range []string{"install", "update", "all"} {
		t.Run(command, func(t *testing.T) {
			home := updateHome(t)
			selectRelease(t, "1.0.0")
			require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, quietLifecycleOptions()))
			require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts-1.0.0"}, quietLifecycleOptions()))
			assert.Equal(t, "sts", readChoices(t).DefaultToolchain)
			selectRelease(t, "2.0.0")
			var err error
			switch command {
			case "install":
				err = Install(t.Context(), InstallRequest{Toolchain: "sts"}, quietLifecycleOptions())
			case "update":
				_, err = UpdateInstalled(t.Context(), parse(t, "sts"), quietLifecycleOptions())
			case "all":
				_, err = UpdateAll(t.Context(), quietLifecycleOptions())
			}
			require.NoError(t, err)
			assert.Equal(t, "sts-2.0.0", readRelease(t, home, "sts").Release)
			assert.Equal(t, "sts-1.0.0", readRelease(t, home, "sts-1.0.0").Release)
			assert.NoDirExists(t, filepath.Join(home, "toolchains", "sts-2.0.0"))
		})
	}
}

func TestChannelAndFixedUninstallAreIndependent(t *testing.T) {
	for _, remove := range []string{"sts", "sts-1.0.0"} {
		t.Run(remove, func(t *testing.T) {
			home := updateHome(t)
			selectRelease(t, "1.0.0")
			require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, quietLifecycleOptions()))
			require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts-1.0.0"}, quietLifecycleOptions()))
			require.NoError(t, RemoveToolchain(remove))
			require.NoError(t, toolchain.RecoverHome())
			assert.NoDirExists(t, filepath.Join(home, "toolchains", remove))
			other := "sts"
			if remove == other {
				other = "sts-1.0.0"
			}
			assert.DirExists(t, filepath.Join(home, "toolchains", other))
		})
	}
}

func TestTrackingTargetsUpdateIndependentlyOfFixedVariant(t *testing.T) {
	home := updateHome(t)
	selectRelease(t, "1.0.0", "ohos")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts", Targets: []string{"ohos"}}, quietLifecycleOptions()))
	tuple, err := sdktarget.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	fixed := "sts-1.0.0-" + tuple
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: fixed}, quietLifecycleOptions()))
	selectRelease(t, "2.0.0", "ohos")
	_, err = UpdateInstalled(t.Context(), parse(t, "sts"), quietLifecycleOptions())
	require.NoError(t, err)
	assert.Equal(t, "sts-2.0.0-"+tuple, readRelease(t, home, "sts-"+tuple).Release)
	assert.Equal(t, fixed, readRelease(t, home, fixed).Release)
	require.NoError(t, RemoveToolchain("sts"))
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "sts-"+tuple))
	assert.DirExists(t, filepath.Join(home, "toolchains", fixed))
}

func TestRemovalCanUnlinkBrokenCustomSDK(t *testing.T) {
	home := updateHome(t)
	link := filepath.Join(home, "toolchains", "custom-sdk")
	if err := os.Symlink(filepath.Join(t.TempDir(), "missing"), link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	require.NoError(t, RemoveToolchain("custom-sdk"))
	_, err := os.Lstat(link)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestSameReleaseHardlinksPayloadButKeepsMetadataIndependent(t *testing.T) {
	for _, mode := range []string{"hardlink", "copy"} {
		t.Run(mode, func(t *testing.T) {
			home := updateHome(t)
			selectRelease(t, "1.0.0")
			sf, err := config.DefaultSettingsFile()
			require.NoError(t, err)
			_, err = sf.Update(config.SettingsUpdate{LinkMode: &mode})
			require.NoError(t, err)
			require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, quietLifecycleOptions()))
			require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts-1.0.0"}, quietLifecycleOptions()))
			for _, rel := range []string{filepath.Join("bin", sdktools.PlatformBinaryName("cjc")), filepath.Join(".cjv", "toolchain.toml")} {
				a, err := os.Stat(filepath.Join(home, "toolchains", "sts", rel))
				require.NoError(t, err)
				b, err := os.Stat(filepath.Join(home, "toolchains", "sts-1.0.0", rel))
				require.NoError(t, err)
				assert.Equal(t, mode == "hardlink" && filepath.Base(rel) != "toolchain.toml", os.SameFile(a, b), rel)
			}
			require.NoError(t, RemoveToolchain("sts"))
			assert.FileExists(t, filepath.Join(home, "toolchains", "sts-1.0.0", "bin", sdktools.PlatformBinaryName("cjc")))
		})
	}
}

func TestChangedInstalledPayloadCannotContaminateFreshInstallation(t *testing.T) {
	home := updateHome(t)
	selectRelease(t, "1.0.0")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, quietLifecycleOptions()))
	compiler := filepath.Join(home, "toolchains", "sts", "bin", sdktools.PlatformBinaryName("cjc"))
	data, err := os.ReadFile(compiler)
	require.NoError(t, err)
	changed := append([]byte(nil), data...)
	changed[0] ^= 1
	require.NoError(t, os.WriteFile(compiler, changed, 0o755))
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts-1.0.0"}, quietLifecycleOptions()))
	fresh, err := os.ReadFile(filepath.Join(home, "toolchains", "sts-1.0.0", "bin", sdktools.PlatformBinaryName("cjc")))
	require.NoError(t, err)
	assert.Equal(t, data, fresh)
}

func TestStaleDistributionCannotReplaceNewerChannel(t *testing.T) {
	home := updateHome(t)
	selectRelease(t, "1.0.0")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, quietLifecycleOptions()))
	stale, err := OpenDistribution(quietLifecycleOptions())
	require.NoError(t, err)
	rt, err := stale.Resolve(t.Context(), parse(t, "sts"), "")
	require.NoError(t, err)
	selectRelease(t, "2.0.0")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, quietLifecycleOptions()))
	require.Error(t, installSelected(t.Context(), stale, rt, true, false, false, quietLifecycleOptions()))
	assert.Equal(t, "sts-2.0.0", readRelease(t, home, "sts").Release)
}

func TestAlreadyInstalledSDKCanBecomeFirstDefault(t *testing.T) {
	for _, identity := range []string{"sts", "sts-1.0.0"} {
		t.Run(identity, func(t *testing.T) {
			home := updateHome(t)
			selectRelease(t, "1.0.0")
			require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: identity}, quietLifecycleOptions()))
			sf, err := config.DefaultSettingsFile()
			require.NoError(t, err)
			empty := ""
			_, err = sf.Update(config.SettingsUpdate{DefaultToolchain: &empty})
			require.NoError(t, err)
			require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: identity}, quietLifecycleOptions()))
			assert.Equal(t, identity, readChoices(t).DefaultToolchain)
			assert.DirExists(t, filepath.Join(home, "toolchains", identity))
		})
	}
}

func TestGroupKeepsLegacyFixedPayloadWithoutForce(t *testing.T) {
	home := updateHome(t)
	selectRelease(t, "1.0.0", "ohos")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts-1.0.0"}, quietLifecycleOptions()))
	fixed := filepath.Join(home, "toolchains", "sts-1.0.0")
	require.NoError(t, os.Remove(filepath.Join(fixed, ".cjv", "toolchain.toml")))
	marker := filepath.Join(fixed, "user-preserved.txt")
	require.NoError(t, os.WriteFile(marker, []byte("legacy custom content"), 0644))
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts-1.0.0", Targets: []string{"ohos"}}, quietLifecycleOptions()))
	require.FileExists(t, marker, "non-force target addition must preserve existing fixed SDK")
}

func TestUpdateAllFindsExplicitHostAlias(t *testing.T) {
	home := updateHome(t)
	tuple, err := sdktarget.CurrentHostTuple("")
	require.NoError(t, err)
	identity := "sts-" + tuple
	selectRelease(t, "1.0.0")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: identity}, quietLifecycleOptions()))
	selectRelease(t, "2.0.0")
	_, err = UpdateAll(t.Context(), quietLifecycleOptions())
	require.NoError(t, err)
	dir, err := toolchain.FindInstalled(parse(t, identity))
	require.NoError(t, err)
	require.Equal(t, filepath.Join(home, "toolchains", "sts"), dir)
	require.Equal(t, "sts-2.0.0", readRelease(t, home, "sts").Release)
}
func TestUpdateAllAllowMissingSkipsDroppedGroupTarget(t *testing.T) {
	home := updateHome(t)
	selectRelease(t, "1.0.0", "ohos")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts", Targets: []string{"ohos"}}, Options{}))
	tuple, err := sdktarget.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	selectRelease(t, "2.0.0")
	report, err := UpdateAll(t.Context(), Options{AllowMissing: true})
	t.Logf("report=%+v error=%v", report, err)
	require.Equal(t, "sts-2.0.0", readRelease(t, home, "sts").Release)
	require.NoDirExists(t, filepath.Join(home, "toolchains", "sts-"+tuple))
	require.NoError(t, err, "group dropped unavailable target as requested; must not process its stale list entry")
}
