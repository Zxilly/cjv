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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows"
)

func TestUpgradeKeepsPublishedReplacementWhenSettingsRestoreFails(t *testing.T) {
	f := newUpgradeFixture(t, false, false)
	// Prevent retiring the old SDK. The settings publication still succeeds,
	// then we lock its real file against replacement before rollback starts.
	t.Chdir(f.oldRoots.TcDir)
	completed := make(chan error, 1)
	go func() {
		_, err := f.upgrade(t, lifecycle.Options{})
		completed <- err
	}()
	deadline := time.Now().Add(10 * time.Second)
	for {
		settings, err := config.LoadSettings(f.sf.Path())
		if err == nil && settings.DefaultToolchain == f.newName {
			break
		}
		select {
		case err := <-completed:
			t.Fatalf("upgrade ended before publishing references: %v", err)
		default:
		}
		if time.Now().After(deadline) {
			t.Fatal("upgrade did not publish references")
		}
		time.Sleep(time.Millisecond)
	}
	path, err := windows.UTF16PtrFromString(f.sf.Path())
	require.NoError(t, err)
	handle, err := windows.CreateFile(path, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, 0, 0)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, windows.CloseHandle(handle)) })
	select {
	case err = <-completed:
	case <-time.After(10 * time.Second):
		t.Fatal("upgrade did not report blocked settings restoration")
	}
	require.Error(t, err)
	assert.ErrorContains(t, err, "restore toolchain settings")
	f.sf.Invalidate()
	settings, err := f.sf.Load()
	require.NoError(t, err)
	assert.Equal(t, f.newName, settings.DefaultToolchain)
	assert.Equal(t, f.newName, settings.Overrides[filepath.Join(f.home, "project")])
	assert.FileExists(t, compilerPath(f.newRoots.TcDir))
	components, err := component.ListInstalled(f.newRoots.TcDir)
	require.NoError(t, err)
	assert.ElementsMatch(t, component.KnownComponents(), components,
		"retaining the replacement must also retain its migrated components")
	for _, path := range []string{f.oldRoots.TcDir, f.oldRoots.StdxDir, f.oldRoots.DocsDir} {
		assert.DirExists(t, path)
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
	assert.Empty(t, settings.DefaultToolchain, "references must be cleared before any content moves")
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
