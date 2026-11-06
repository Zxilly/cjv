//go:build windows

package selfcmd

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/fsops"
	"github.com/Zxilly/cjv/internal/i18n"
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
	i18n.Init("en")
	t.Cleanup(func() { i18n.Init("en") })
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

func TestDetachedUninstallPreservesReinstallationAtOriginalPath(t *testing.T) {
	parent := t.TempDir()
	home := filepath.Join(parent, "home")
	managed := filepath.Join(home, "bin", "cjv.exe")
	require.NoError(t, os.MkdirAll(filepath.Dir(managed), 0o755))
	require.NoError(t, os.WriteFile(managed, []byte("old executable"), 0o755))
	oldCurrent, oldStart := currentExecutablePath, startDetachedUninstallCleanup
	t.Cleanup(func() { currentExecutablePath, startDetachedUninstallCleanup = oldCurrent, oldStart })
	currentExecutablePath = func() (string, error) { return managed, nil }
	var cleanupPath string
	startDetachedUninstallCleanup = func(path string) error { cleanupPath = path; return nil }
	require.NoError(t, removeHomeDir(home, managed))

	// A fresh installation completes before the detached cleanup runs.
	require.NoError(t, os.MkdirAll(filepath.Dir(managed), 0o755))
	require.NoError(t, os.WriteFile(managed, []byte("new executable"), 0o755))
	require.NotEmpty(t, cleanupPath)
	require.Equal(t, parent, filepath.Dir(cleanupPath), "cleanup must stay in this test's temporary parent")
	require.NoError(t, os.RemoveAll(cleanupPath))
	data, err := os.ReadFile(managed)
	require.NoError(t, err, "old cleanup must not delete the new installation")
	require.Equal(t, "new executable", string(data))
}

func TestDetachedUninstallOfRunningExecutable(t *testing.T) {
	if home := os.Getenv("CJV_TEST_UNINSTALL_HOME"); home != "" {
		// Record the destination, but let the parent test control when cleanup
		// starts so the reinstallation/cleanup ordering is deterministic.
		startDetachedUninstallCleanup = func(path string) error {
			return os.WriteFile(os.Getenv("CJV_TEST_UNINSTALL_RECORD"), []byte(path), 0o600)
		}
		managed := filepath.Join(home, "bin", "cjv.exe")
		require.NoError(t, removeHomeDir(home, managed))
		require.Eventually(t, func() bool {
			_, err := os.Stat(os.Getenv("CJV_TEST_UNINSTALL_RECORD") + ".release")
			return err == nil
		}, 15*time.Second, 10*time.Millisecond)
		return
	}
	parent := t.TempDir()
	home := filepath.Join(parent, "home")
	managed := filepath.Join(home, "bin", "cjv.exe")
	record := filepath.Join(parent, "cleanup-path")
	require.NoError(t, os.MkdirAll(filepath.Dir(managed), 0o755))
	executable, err := os.Executable()
	require.NoError(t, err)
	data, err := os.ReadFile(executable)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(managed, data, 0o755))
	external := filepath.Join(parent, "external-sdk")
	require.NoError(t, os.MkdirAll(external, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(external, "keep"), []byte("user SDK"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains"), 0o755))
	require.NoError(t, fsops.SymlinkOrJunction(external, filepath.Join(home, "toolchains", "custom")))
	child := exec.CommandContext(t.Context(), managed, "-test.run=^TestDetachedUninstallOfRunningExecutable$")
	child.Env = append(os.Environ(), "CJV_TEST_UNINSTALL_HOME="+home, "CJV_TEST_UNINSTALL_RECORD="+record)
	var output bytes.Buffer
	child.Stdout, child.Stderr = &output, &output
	require.NoError(t, child.Start())
	t.Cleanup(func() { _ = child.Process.Kill() })
	childDone := make(chan error, 1)
	go func() { childDone <- child.Wait() }()
	require.Eventually(t, func() bool {
		_, err := os.Stat(record)
		return err == nil
	}, 10*time.Second, 10*time.Millisecond)
	retiredBytes, err := os.ReadFile(record)
	require.NoError(t, err)
	retired := string(retiredBytes)
	require.Equal(t, parent, filepath.Dir(retired))
	require.NoDirExists(t, home)
	require.FileExists(t, filepath.Join(retired, "home", "bin", "cjv.exe"))
	require.NoError(t, os.MkdirAll(filepath.Dir(managed), 0o755))
	require.NoError(t, os.WriteFile(managed, []byte("new installation"), 0o755))
	cleanup := uninstallCleanupCommand(retired, child.Process.Pid)
	var cleanupOutput bytes.Buffer
	cleanup.Stdout, cleanup.Stderr = &cleanupOutput, &cleanupOutput
	require.NoError(t, cleanup.Start())
	t.Cleanup(func() { _ = cleanup.Process.Kill() })
	cleanupDone := make(chan error, 1)
	go func() { cleanupDone <- cleanup.Wait() }()
	select {
	case err := <-cleanupDone:
		t.Fatalf("cleanup returned before the owner exited: %v\n%s", err, cleanupOutput.String())
	case <-time.After(3500 * time.Millisecond):
		// Outlive the old hard-coded three-second delay.
	}
	require.FileExists(t, filepath.Join(retired, "home", "bin", "cjv.exe"))
	require.NoError(t, os.WriteFile(record+".release", nil, 0o600))
	require.NoError(t, <-childDone, "%s", output.String())
	require.NoError(t, <-cleanupDone, "%s", cleanupOutput.String())
	require.NoDirExists(t, retired)
	current, err := os.ReadFile(managed)
	require.NoError(t, err)
	require.Equal(t, "new installation", string(current))
	require.FileExists(t, filepath.Join(external, "keep"), "cleanup must not follow linked SDKs")
}

func TestDetachedUninstallStartFailurePreservesConcurrentReinstall(t *testing.T) {
	parent := t.TempDir()
	home := filepath.Join(parent, "home")
	managed := filepath.Join(home, "bin", "cjv.exe")
	require.NoError(t, os.MkdirAll(filepath.Dir(managed), 0o755))
	require.NoError(t, os.WriteFile(managed, []byte("old executable"), 0o755))
	oldCurrent, oldStart := currentExecutablePath, startDetachedUninstallCleanup
	t.Cleanup(func() { currentExecutablePath, startDetachedUninstallCleanup = oldCurrent, oldStart })
	currentExecutablePath = func() (string, error) { return managed, nil }
	var retired string
	startDetachedUninstallCleanup = func(path string) error {
		retired = path
		require.NoError(t, os.MkdirAll(filepath.Dir(managed), 0o755))
		require.NoError(t, os.WriteFile(managed, []byte("new executable"), 0o755))
		return errors.New("start failed")
	}
	require.Error(t, removeHomeDir(home, managed))
	data, err := os.ReadFile(managed)
	require.NoError(t, err)
	require.Equal(t, "new executable", string(data))
	data, err = os.ReadFile(filepath.Join(retired, "home", "bin", "cjv.exe"))
	require.NoError(t, err)
	require.Equal(t, "old executable", string(data), "keep retired data when rollback is obstructed")
}
