//go:build windows

package lifecycle_test

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
)

func TestFirstInstallKeepsSDKWhenPublishedSettingsCannotBeRestored(t *testing.T) {
	for _, name := range []string{"lts", "lts-1.0.5"} {
		t.Run(name, func(t *testing.T) {
			home, sf, _ := prepareInstallTest(t)
			failure := errors.New("failed after publishing default")
			lifecycle.SetAfterPublishHook(t, func() error {
				path, err := windows.UTF16PtrFromString(sf.Path())
				require.NoError(t, err)
				handle, err := windows.CreateFile(path, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, 0, 0)
				require.NoError(t, err)
				t.Cleanup(func() { require.NoError(t, windows.CloseHandle(handle)) })
				return failure
			})
			err := lifecycle.Install(t.Context(), lifecycle.InstallRequest{Toolchain: name}, lifecycle.Options{})
			require.ErrorIs(t, err, failure)
			require.ErrorContains(t, err, "restore toolchain settings")
			settings, err := config.LoadSettings(sf.Path())
			require.NoError(t, err)
			assert.Equal(t, name, settings.DefaultToolchain)
			assert.FileExists(t, compilerPath(filepath.Join(home, "toolchains", name)))
			require.NoError(t, toolchain.RecoverHome())
			assert.FileExists(t, compilerPath(filepath.Join(home, "toolchains", name)))
		})
	}
}

func TestRetirementCrashHelper(t *testing.T) {
	home := os.Getenv("CJV_TEST_RETIRE_CRASH_HOME")
	if home == "" {
		t.Skip("subprocess helper")
	}
	config.IsolateForTest(t, home)
	name := "lts-1.0.0"
	roots, err := component.RootsFor(name)
	require.NoError(t, err)
	// Keep the last removal retrying long enough to interrupt after the two
	// external roots have actually moved into the on-disk transaction.
	require.NoError(t, os.Chdir(roots.TcDir))
	go func() {
		for {
			_, docsErr := os.Stat(roots.DocsDir)
			_, stdxErr := os.Stat(roots.StdxDir)
			if errors.Is(docsErr, os.ErrNotExist) && errors.Is(stdxErr, os.ErrNotExist) {
				os.Exit(42)
			}
			time.Sleep(time.Millisecond)
		}
	}()
	_ = lifecycle.RemoveToolchain(name)
	os.Exit(43)
}

func TestRemovalRecoversRealCrashAfterReferencePublication(t *testing.T) {
	f := newUpgradeFixture(t, false, false)
	executable, err := os.Executable()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, "-test.run=^TestRetirementCrashHelper$")
	cmd.Env = append(os.Environ(), "CJV_TEST_RETIRE_CRASH_HOME="+f.home)
	output, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr, "%s", output)
	require.Equal(t, 42, exitErr.ExitCode(), "%s", output)
	assert.NoDirExists(t, f.oldRoots.StdxDir)
	assert.NoDirExists(t, f.oldRoots.DocsDir)
	settings, err := config.LoadSettings(f.sf.Path())
	require.NoError(t, err)
	assert.NotEqual(t, f.oldName, settings.DefaultToolchain, "removed identity must no longer be selected")
	// The direct production retry recovers before checking whether the source
	// exists. Its preparation step restores inspectable original content.
	require.NoError(t, lifecycle.PrepareToolchainRemoval(f.oldName))
	for _, path := range []string{f.oldRoots.TcDir, f.oldRoots.StdxDir, f.oldRoots.DocsDir} {
		assert.DirExists(t, path)
	}
	f.sf.Invalidate()
	require.NoError(t, lifecycle.RemoveToolchain(f.oldName))
	for _, path := range []string{f.oldRoots.TcDir, f.oldRoots.StdxDir, f.oldRoots.DocsDir} {
		assert.NoDirExists(t, path)
	}
}
