//go:build windows

package selfmgmt

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRemoveHomeDirDeletesHomeWhenManagedExeIsNotCurrentProcess(t *testing.T) {
	home := t.TempDir()
	managedExe := filepath.Join(home, "bin", "cjv.exe")
	require.NoError(t, os.MkdirAll(filepath.Dir(managedExe), 0o755))
	require.NoError(t, os.WriteFile(managedExe, []byte("exe"), 0o755))

	oldCurrent := currentExecutablePath
	currentExecutablePath = func() (string, error) {
		return filepath.Join(t.TempDir(), "other.exe"), nil
	}
	t.Cleanup(func() { currentExecutablePath = oldCurrent })

	require.NoError(t, removeHomeDir(home, managedExe))

	assert.NoDirExists(t, home)
}

func TestRemoveHomeDirRejectsDangerousAndMissingManagedPath(t *testing.T) {
	driveRoot := filepath.VolumeName(t.TempDir()) + `\`
	require.Error(t, removeHomeDir(driveRoot, filepath.Join(driveRoot, "bin", "cjv.exe")))

	home := t.TempDir()
	err := removeHomeDir(home, filepath.Join(home, "bin", "missing.exe"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "managed binary not found")
}

func TestRemoveHomeDirRestoresManagedExeWhenDetachedCleanupFails(t *testing.T) {
	home := t.TempDir()
	binDir := filepath.Join(home, "bin")
	managedExe := filepath.Join(binDir, "cjv.exe")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	require.NoError(t, os.WriteFile(managedExe, []byte("exe"), 0o755))

	oldCurrent := currentExecutablePath
	oldStart := startDetachedUninstallCleanup
	currentExecutablePath = func() (string, error) { return managedExe, nil }
	startDetachedUninstallCleanup = func(home string) error { return errors.New("start failed") }
	t.Cleanup(func() {
		currentExecutablePath = oldCurrent
		startDetachedUninstallCleanup = oldStart
	})

	err := removeHomeDir(home, managedExe)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "start failed")
	assert.FileExists(t, managedExe)
}

func TestRemoveSymlinksHandlesMissingDirectory(t *testing.T) {
	require.NoError(t, removeSymlinks(filepath.Join(t.TempDir(), "missing")))
}
