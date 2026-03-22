package toolchain

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for CleanupStagingDirs -- recovery from interrupted installations.
//
// When "cjv install" is interrupted (Ctrl-C, power loss, etc.), it can
// leave behind .staging (incomplete new install) and .old (backup of
// previous install) directories. CleanupStagingDirs must restore a
// usable state.

func TestCleanupStagingDirs_RemovesAbandonedStaging(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CJV_HOME", tmpDir)

	tcDir := filepath.Join(tmpDir, "toolchains")
	staging := filepath.Join(tcDir, "lts-1.0.6.staging")
	require.NoError(t, os.MkdirAll(staging, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(staging, "partial.bin"), []byte("x"), 0o644))

	CleanupStagingDirs()

	_, err := os.Stat(staging)
	assert.True(t, os.IsNotExist(err), "incomplete staging directory should be removed")
}

func TestCleanupStagingDirs_RestoresBackupWhenOriginalMissing(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CJV_HOME", tmpDir)

	tcDir := filepath.Join(tmpDir, "toolchains")
	backup := filepath.Join(tcDir, "sts-2.0.0.old")
	require.NoError(t, os.MkdirAll(backup, 0o755))

	CleanupStagingDirs()

	restored := filepath.Join(tcDir, "sts-2.0.0")
	_, err := os.Stat(restored)
	assert.NoError(t, err, "backup should be restored when original is missing")

	_, err = os.Stat(backup)
	assert.True(t, os.IsNotExist(err), ".old should no longer exist after restoration")
}

func TestCleanupStagingDirs_RemovesBackupWhenOriginalExists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CJV_HOME", tmpDir)

	tcDir := filepath.Join(tmpDir, "toolchains")
	original := filepath.Join(tcDir, "sts-2.0.0")
	backup := filepath.Join(tcDir, "sts-2.0.0.old")
	require.NoError(t, os.MkdirAll(original, 0o755))
	require.NoError(t, os.MkdirAll(backup, 0o755))

	CleanupStagingDirs()

	_, err := os.Stat(original)
	assert.NoError(t, err, "current install should not be touched")

	_, err = os.Stat(backup)
	assert.True(t, os.IsNotExist(err), "obsolete backup should be removed")
}

func TestCleanupStagingDirs_LeavesNormalToolchainsAlone(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CJV_HOME", tmpDir)

	tcDir := filepath.Join(tmpDir, "toolchains")
	require.NoError(t, os.MkdirAll(filepath.Join(tcDir, "lts-1.0.5"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tcDir, "sts-2.0.0"), 0o755))

	CleanupStagingDirs()

	assert.DirExists(t, filepath.Join(tcDir, "lts-1.0.5"))
	assert.DirExists(t, filepath.Join(tcDir, "sts-2.0.0"))
}

func TestCleanupStagingDirs_NoToolchainsDirIsNotAnError(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CJV_HOME", tmpDir)

	assert.NotPanics(t, func() { CleanupStagingDirs() })
}
