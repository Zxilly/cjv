package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
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

func TestTrackingInstallAndUpdateReplaceOnlyChannelOwnership(t *testing.T) {
	for _, command := range []string{"install", "update", "all"} {
		t.Run(command, func(t *testing.T) {
			home := updateHome(t)
			selectRelease(t, "1.0.0")
			require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, quietLifecycleOptions()))
			assert.Equal(t, "sts", readChoices(t).DefaultToolchain)
			selectRelease(t, "2.0.0")
			switch command {
			case "install":
				require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, quietLifecycleOptions()))
			case "update":
				_, err := UpdateInstalled(t.Context(), parse(t, "sts"), quietLifecycleOptions())
				require.NoError(t, err)
			case "all":
				_, err := UpdateAll(t.Context(), quietLifecycleOptions())
				require.NoError(t, err)
			}
			assert.NoDirExists(t, filepath.Join(home, "toolchains", "sts-1.0.0"))
			assert.DirExists(t, filepath.Join(home, "toolchains", "sts-2.0.0"))
			assert.Equal(t, "sts-2.0.0", readChoices(t).Installations["sts"])
			dir, err := installedForChannel(parse(t, "sts").Channel)
			require.NoError(t, err)
			assert.Equal(t, "sts-2.0.0", dir)
		})
	}
}

func TestFixedAndChannelCanShareReleaseAndUninstallIndependently(t *testing.T) {
	for _, remove := range []string{"sts", "sts-1.0.0"} {
		t.Run(remove, func(t *testing.T) {
			home := updateHome(t)
			selectRelease(t, "1.0.0")
			require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, quietLifecycleOptions()))
			require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts-1.0.0"}, quietLifecycleOptions()))
			require.NoError(t, RemoveToolchain(remove))
			assert.DirExists(t, filepath.Join(home, "toolchains", "sts-1.0.0"))
			assert.NotContains(t, readChoices(t).Installations, remove)
			if remove == "sts" {
				_, err := UpdateAll(t.Context(), quietLifecycleOptions())
				require.NoError(t, err)
				assert.Equal(t, "sts-1.0.0", readChoices(t).Installations["sts-1.0.0"])
			} else {
				selectRelease(t, "2.0.0")
				_, err := UpdateAll(t.Context(), quietLifecycleOptions())
				require.NoError(t, err)
				assert.NoDirExists(t, filepath.Join(home, "toolchains", "sts-1.0.0"))
			}
		})
	}
}

func TestFixedVersionSurvivesChannelUpdate(t *testing.T) {
	home := updateHome(t)
	selectRelease(t, "1.0.0")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts-1.0.0"}, quietLifecycleOptions()))
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, quietLifecycleOptions()))
	selectRelease(t, "2.0.0")
	_, err := UpdateAll(t.Context(), quietLifecycleOptions())
	require.NoError(t, err)
	assert.DirExists(t, filepath.Join(home, "toolchains", "sts-1.0.0"))
	assert.DirExists(t, filepath.Join(home, "toolchains", "sts-2.0.0"))
	assert.Equal(t, "sts-1.0.0", readChoices(t).DefaultToolchain)
	assert.Equal(t, "sts-2.0.0", readChoices(t).Installations["sts"])
}

func TestLegacyInstallsRemainFixedAndChannelUpdateInstallsMissing(t *testing.T) {
	home := updateHome(t)
	fakeInstalled(t, home, "sts-1.0.0")
	selectRelease(t, "2.0.0")
	report, err := UpdateAll(t.Context(), quietLifecycleOptions())
	require.NoError(t, err)
	assert.Empty(t, applied(report))
	assert.Equal(t, "sts-1.0.0", readChoices(t).Installations["sts-1.0.0"])
	_, err = UpdateInstalled(t.Context(), parse(t, "sts"), quietLifecycleOptions())
	require.NoError(t, err)
	assert.DirExists(t, filepath.Join(home, "toolchains", "sts-1.0.0"))
	assert.Equal(t, "sts-2.0.0", readChoices(t).Installations["sts"])
}

func TestTrackingTargetUpdatesButExplicitVariantRemainsPinned(t *testing.T) {
	home := updateHome(t)
	selectRelease(t, "1.0.0", "ohos")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts", Targets: []string{"ohos"}}, quietLifecycleOptions()))
	tuple, err := sdktarget.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	old := "sts-1.0.0-" + tuple
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: old}, quietLifecycleOptions()))
	selectRelease(t, "2.0.0", "ohos")
	_, err = UpdateInstalled(t.Context(), parse(t, "sts"), quietLifecycleOptions())
	require.NoError(t, err)
	assert.DirExists(t, filepath.Join(home, "toolchains", old))
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "sts-1.0.0"))
	assert.Equal(t, "sts-2.0.0-"+tuple, readChoices(t).Installations["sts/"+tuple])
}

func TestTrackingSelectionIgnoresNewerFixedRelease(t *testing.T) {
	updateHome(t)
	selectRelease(t, "1.0.0")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, quietLifecycleOptions()))
	selectRelease(t, "3.0.0")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts-3.0.0"}, quietLifecycleOptions()))
	dir, err := toolchain.FindInstalled(parse(t, "sts"))
	require.NoError(t, err)
	assert.Equal(t, "sts-1.0.0", filepath.Base(dir))
	sf, err := config.DefaultSettingsFile()
	require.NoError(t, err)
	sf.Invalidate()
	dir, err = toolchain.FindInstalled(parse(t, "sts"))
	require.NoError(t, err)
	assert.Equal(t, "sts-1.0.0", filepath.Base(dir))
}

func TestChannelUninstallRemovesTargetsButPreservesPinnedVariant(t *testing.T) {
	home := updateHome(t)
	selectRelease(t, "1.0.0", "ohos", "android")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts", Targets: []string{"ohos", "android"}}, quietLifecycleOptions()))
	ohos, err := sdktarget.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	android, err := sdktarget.CurrentTargetTuple("", "android")
	require.NoError(t, err)
	pinned := "sts-1.0.0-" + ohos
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: pinned}, quietLifecycleOptions()))
	require.NoError(t, RemoveToolchain("sts"))
	assert.DirExists(t, filepath.Join(home, "toolchains", pinned))
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "sts-1.0.0-"+android))
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "sts-1.0.0"))
	assert.Equal(t, map[string]string{pinned: pinned}, readChoices(t).Installations)
	_, err = toolchain.FindInstalled(parse(t, "sts"))
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestUninstallUnpinnedVersionCannotRemoveTrackingSDK(t *testing.T) {
	home := updateHome(t)
	selectRelease(t, "1.0.0")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, quietLifecycleOptions()))
	require.Error(t, RemoveToolchain("sts-1.0.0"))
	assert.Equal(t, "sts-1.0.0", readChoices(t).Installations["sts"])
	assert.DirExists(t, filepath.Join(home, "toolchains", "sts-1.0.0"))
}

func TestInstallPublicationDoesNotRestoreRemovedIdentities(t *testing.T) {
	updateHome(t)
	selectRelease(t, "1.0.0")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, quietLifecycleOptions()))
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts-1.0.0"}, quietLifecycleOptions()))
	stale, err := OpenDistribution(quietLifecycleOptions())
	require.NoError(t, err)
	require.NoError(t, recordLegacyInstallations(t.Context(), stale, false))
	require.NoError(t, RemoveToolchain("sts-1.0.0"))
	selectRelease(t, "2.0.0")
	fresh, err := OpenDistribution(quietLifecycleOptions())
	require.NoError(t, err)
	rt, err := fresh.Resolve(t.Context(), parse(t, "sts"), "")
	require.NoError(t, err)
	require.NoError(t, installSelected(t.Context(), stale, rt, false, false, false, quietLifecycleOptions()))
	assert.NotContains(t, readChoices(t).Installations, "sts-1.0.0")
	require.NoError(t, RemoveToolchain("sts"))
	// Publication into an existing SDK uses the same live merge rules.
	require.NoError(t, installSelected(t.Context(), stale, rt, false, false, false, quietLifecycleOptions()))
	assert.Equal(t, map[string]string{"sts-2.0.0": "sts-2.0.0"}, readChoices(t).Installations)
}

func TestStaleChannelInstallationCannotOverwriteNewerSelection(t *testing.T) {
	updateHome(t)
	selectRelease(t, "1.0.0")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, quietLifecycleOptions()))
	stale, err := OpenDistribution(quietLifecycleOptions())
	require.NoError(t, err)
	require.NoError(t, recordLegacyInstallations(t.Context(), stale, false))
	rt, err := stale.Resolve(t.Context(), parse(t, "sts"), "")
	require.NoError(t, err)
	selectRelease(t, "2.0.0")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, quietLifecycleOptions()))
	require.Error(t, installSelected(t.Context(), stale, rt, true, false, true, quietLifecycleOptions()))
	assert.Equal(t, "sts-2.0.0", readChoices(t).Installations["sts"])
	dir, err := toolchain.FindInstalled(parse(t, "sts"))
	require.NoError(t, err)
	assert.Equal(t, "sts-2.0.0", filepath.Base(dir))
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
