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

func TestCleanupStagingDirs_RestoresFstxBackupWhenOriginalMissing(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CJV_HOME", tmpDir)

	tcDir := filepath.Join(tmpDir, "toolchains")
	backup := filepath.Join(tcDir, ".fstx-crash", "0-lts-1.0.5")
	require.NoError(t, os.MkdirAll(backup, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(backup, "release.txt"), []byte("old"), 0o644))

	CleanupStagingDirs()

	restored := filepath.Join(tcDir, "lts-1.0.5")
	assert.FileExists(t, filepath.Join(restored, "release.txt"))
	assert.NoDirExists(t, filepath.Join(tcDir, ".fstx-crash"))
}

func TestCleanupStagingDirs_PreservesBackupWhenOriginalExists(t *testing.T) {
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

	assert.DirExists(t, backup, "without a commit marker the backup may be the only good SDK")
}

func TestCleanupStagingDirs_PreservesAmbiguousTransactionsAndStaging(t *testing.T) {
	for _, artifact := range []string{"legacy-conflict", "unknown-file", "invalid-journal"} {
		t.Run(artifact, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("CJV_HOME", home)
			root := filepath.Join(home, "toolchains")
			txDir := filepath.Join(root, ".fstx-interrupted")
			backup := filepath.Join(txDir, "0-lts-1.0.5")
			require.NoError(t, os.MkdirAll(backup, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(backup, "old-sdk"), []byte("previous SDK"), 0o644))
			staging := filepath.Join(root, "lts-1.0.5.staging")
			require.NoError(t, os.MkdirAll(staging, 0o755))
			switch artifact {
			case "legacy-conflict":
				require.NoError(t, os.MkdirAll(filepath.Join(root, "lts-1.0.5"), 0o755))
			case "unknown-file":
				require.NoError(t, os.WriteFile(filepath.Join(txDir, "unknown"), []byte("retain me"), 0o644))
			case "invalid-journal":
				require.NoError(t, os.WriteFile(filepath.Join(txDir, "journal.json"), []byte("{"), 0o644))
			}

			CleanupStagingDirs()

			assert.DirExists(t, txDir)
			assert.DirExists(t, staging)
			// Legacy recovery may restore an unambiguous backup before finding
			// an unknown entry, but must keep the old SDK in either location.
			_, backupErr := os.Stat(filepath.Join(backup, "old-sdk"))
			_, restoredErr := os.Stat(filepath.Join(root, "lts-1.0.5", "old-sdk"))
			assert.True(t, backupErr == nil || restoredErr == nil)
		})
	}
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

func TestCleanupStagingDirs_RestoresDotPrefixedLegacyBackup(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	original := filepath.Join(home, "toolchains", ".local-sdk")
	backup := original + BackupSuffix
	require.NoError(t, os.MkdirAll(backup, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(backup, "old-sdk"), []byte("old SDK"), 0o644))
	CleanupStagingDirs()
	assert.FileExists(t, filepath.Join(original, "old-sdk"))
	assert.NoDirExists(t, backup)
}
