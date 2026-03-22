//go:build windows

package selfupdate

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for CleanupOldBinaries — removes leftover files from previous
// self-update or uninstall operations on Windows.

func TestCleanupOldBinaries_RemovesGCFiles(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	binDir := filepath.Join(home, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))

	// Create the managed binary
	binaryName := proxy.CjvBinaryName()
	require.NoError(t, os.WriteFile(filepath.Join(binDir, binaryName), []byte("cjv"), 0o755))

	// Create garbage-collected leftover files
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "cjv-gc-12345.exe"), []byte("old"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(binDir, "cjv-gc-67890.exe"), []byte("old"), 0o644))

	// Create .old backup file
	require.NoError(t, os.WriteFile(filepath.Join(binDir, ".cjv.exe.old"), []byte("old"), 0o644))

	CleanupOldBinaries()

	// GC files should be removed
	_, err := os.Stat(filepath.Join(binDir, "cjv-gc-12345.exe"))
	assert.True(t, os.IsNotExist(err), "gc files should be cleaned up")

	// .old backup should be removed
	_, err = os.Stat(filepath.Join(binDir, ".cjv.exe.old"))
	assert.True(t, os.IsNotExist(err), "old backup should be cleaned up")

	// The managed binary should still exist
	assert.FileExists(t, filepath.Join(binDir, binaryName))
}

func TestCleanupOldBinaries_NoBinaryNoOp(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	// No binary, no crash
	assert.NotPanics(t, func() { CleanupOldBinaries() })
}
