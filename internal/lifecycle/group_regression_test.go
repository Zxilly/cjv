package lifecycle

import (
	"github.com/Zxilly/cjv/internal/config"
	"os"
	"os/exec"
	"testing"

	"github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/require"
)

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
		config.IsolateForTest(t, os.Getenv(config.EnvHome))
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
