package lifecycle

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/require"
)

func componentGroupFixture(t *testing.T) ([]component.Roots, []os.FileInfo) {
	t.Helper()
	f := newHistoryUpdateFixture(t)
	sf, err := config.DefaultSettingsFile()
	require.NoError(t, err)
	mode := "copy"
	_, err = sf.Update(config.SettingsUpdate{LinkMode: &mode})
	require.NoError(t, err)
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts", Targets: []string{"ohos"}}, Options{}))
	var roots []component.Roots
	var payloads []os.FileInfo
	for _, name := range []string{"sts", "sts-" + f.host + "-ohos"} {
		root, err := component.RootsFor(name)
		require.NoError(t, err)
		payload, err := os.Stat(filepath.Join(root.TcDir, "bin", "cjc"))
		require.NoError(t, err)
		roots, payloads = append(roots, root), append(payloads, payload)
	}
	return roots, payloads
}

func assertComponentGroupSDKsUnchanged(t *testing.T, roots []component.Roots, payloads []os.FileInfo) {
	t.Helper()
	for i, root := range roots {
		current, err := os.Stat(filepath.Join(root.TcDir, "bin", "cjc"))
		require.NoError(t, err)
		require.True(t, os.SameFile(payloads[i], current), "unchanged SDK must stay in place: %s", root.TcDir)
	}
}

func TestComponentOnlyGroupStagesNoSDKAndRollsBackTogether(t *testing.T) {
	roots, payloads := componentGroupFixture(t)
	staged := false
	afterStagingHook = func() error {
		staged = true
		for _, root := range roots {
			require.NoFileExists(t, filepath.Join(config.StagingDir(root.TcDir), "bin", "cjc"))
			require.FileExists(t, filepath.Join(config.StagingDir(root.TcDir), component.MetaDir, "manifest-docs"))
		}
		return nil
	}
	failure := errors.New("group finalization failed")
	afterFinalizeHook = func() error { return failure }
	t.Cleanup(func() { afterStagingHook, afterFinalizeHook = nil, nil })
	err := Install(t.Context(), InstallRequest{Toolchain: "sts", Components: []string{"docs"}}, Options{})
	require.ErrorIs(t, err, failure)
	require.True(t, staged)
	assertComponentGroupSDKsUnchanged(t, roots, payloads)
	for _, root := range roots {
		require.False(t, component.IsInstalled(root.TcDir, component.Docs))
		require.NoFileExists(t, filepath.Join(root.DocsDir, "main", "index.html"))
	}
	afterFinalizeHook = nil
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts", Components: []string{"docs"}}, Options{}))
	assertComponentGroupSDKsUnchanged(t, roots, payloads)
	for _, root := range roots {
		require.True(t, component.IsInstalled(root.TcDir, component.Docs))
		require.FileExists(t, filepath.Join(root.DocsDir, "main", "index.html"))
	}
}

func TestComponentOnlyGroupRecoversRealProcessInterruption(t *testing.T) {
	if os.Getenv("CJV_TEST_COMPONENT_GROUP_CRASH") == "1" {
		configtest.Isolate(t, os.Getenv(config.EnvHome))
		afterFinalizeHook = func() error { os.Exit(73); return nil }
		_ = Install(t.Context(), InstallRequest{Toolchain: "sts", Components: []string{"docs"}}, Options{})
		os.Exit(74)
	}
	roots, payloads := componentGroupFixture(t)
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestComponentOnlyGroupRecoversRealProcessInterruption$")
	cmd.Env = append(os.Environ(), "CJV_TEST_COMPONENT_GROUP_CRASH=1")
	output, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr, "%s", output)
	require.Equal(t, 73, exitErr.ExitCode(), "%s", output)
	for _, root := range roots {
		require.FileExists(t, filepath.Join(root.DocsDir, "main", "index.html"))
	}
	assertComponentGroupSDKsUnchanged(t, roots, payloads)
	require.NoError(t, toolchain.RecoverHome())
	assertComponentGroupSDKsUnchanged(t, roots, payloads)
	for _, root := range roots {
		require.False(t, component.IsInstalled(root.TcDir, component.Docs))
		require.NoFileExists(t, filepath.Join(root.DocsDir, "main", "index.html"))
		require.NoDirExists(t, config.StagingDir(root.TcDir))
		require.NoDirExists(t, config.StagingDir(root.DocsDir))
	}
	require.NoError(t, toolchain.RecoverHome(), "recovery remains idempotent")
}
