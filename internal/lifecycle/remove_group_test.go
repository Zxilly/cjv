package lifecycle

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Zxilly/cjv/internal/target"
	"github.com/stretchr/testify/require"
)

func TestTrackingGroupRemovalRestoresHostWhenLaterTargetIsBusy(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows directory handle blocks target retirement")
	}
	home := updateHome(t)
	selectRelease(t, "1.0.0", "ohos")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts", Targets: []string{"ohos"}}, Options{}))
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts-1.0.0"}, Options{}))
	tuple, err := target.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	before := readChoices(t)
	t.Chdir(filepath.Join(home, "toolchains", "sts-"+tuple))
	err = RemoveToolchain("sts")
	require.Error(t, err)
	require.Equal(t, "sts-1.0.0", readRelease(t, home, "sts").Release)
	require.Equal(t, "sts-1.0.0-"+tuple, readRelease(t, home, "sts-"+tuple).Release)
	require.Equal(t, before, readChoices(t))
	require.NoError(t, os.Chdir(home))
	require.NoError(t, RemoveToolchain("sts"))
	require.NoDirExists(t, filepath.Join(home, "toolchains", "sts"))
	require.NoDirExists(t, filepath.Join(home, "toolchains", "sts-"+tuple))
	require.Equal(t, "sts-1.0.0", readRelease(t, home, "sts-1.0.0").Release)
}
