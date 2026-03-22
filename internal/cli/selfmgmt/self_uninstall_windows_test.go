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

func TestRemoveHomeDirRestoresManagedBinaryWhenCleanupStartFails(t *testing.T) {
	oldCurrentExecutablePath := currentExecutablePath
	oldStartDetachedUninstallCleanup := startDetachedUninstallCleanup
	defer func() {
		currentExecutablePath = oldCurrentExecutablePath
		startDetachedUninstallCleanup = oldStartDetachedUninstallCleanup
	}()

	home := t.TempDir()
	binDir := filepath.Join(home, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	managedExe := filepath.Join(binDir, "cjv.exe")
	require.NoError(t, os.WriteFile(managedExe, []byte("cjv"), 0o755))

	currentExecutablePath = func() (string, error) {
		return managedExe, nil
	}
	startDetachedUninstallCleanup = func(string) error {
		return errors.New("start failed")
	}

	err := removeHomeDir(home, managedExe)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start detached uninstall cleanup")
	assert.FileExists(t, managedExe)

	matches, globErr := filepath.Glob(filepath.Join(binDir, "cjv-gc-*.exe"))
	require.NoError(t, globErr)
	assert.Empty(t, matches)
}
