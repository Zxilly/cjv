package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for cleanDownloadCache -- removes all entries (files and
// subdirectories) from the downloads directory.

func TestCleanDownloadCache_RemovesAll(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	downloads := filepath.Join(home, "downloads")
	require.NoError(t, os.MkdirAll(downloads, 0o755))

	// Create files that should be cleaned
	require.NoError(t, os.WriteFile(filepath.Join(downloads, "sdk-1.0.5.zip"), []byte("data"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(downloads, "sdk-2.0.0.tar.gz"), []byte("data"), 0o644))

	// Create a subdirectory that should also be removed
	require.NoError(t, os.MkdirAll(filepath.Join(downloads, "partial"), 0o755))

	removed, err := cleanDownloadCache()
	require.NoError(t, err)

	// All entries should be removed
	assert.Equal(t, 3, removed)

	_, err = os.Stat(filepath.Join(downloads, "sdk-1.0.5.zip"))
	assert.True(t, os.IsNotExist(err), "archive files should be removed")

	_, err = os.Stat(filepath.Join(downloads, "partial"))
	assert.True(t, os.IsNotExist(err), "subdirectories should also be removed")
}

func TestCleanDownloadCache_EmptyOrMissingDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	// No downloads directory — should not panic
	assert.NotPanics(t, func() {
		removed, err := cleanDownloadCache()
		require.NoError(t, err)
		assert.Zero(t, removed)
	})
}

func TestCleanDownloadCache_RemovesReadonlyEntry(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("readonly file deletion is Windows-specific")
	}

	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	cacheFile := filepath.Join(home, "downloads", "archive.zip")
	require.NoError(t, os.MkdirAll(filepath.Dir(cacheFile), 0o755))
	require.NoError(t, os.WriteFile(cacheFile, []byte("data"), 0o644))
	require.NoError(t, os.Chmod(cacheFile, 0o444))

	removed, err := cleanDownloadCache()

	require.NoError(t, err)
	assert.Equal(t, 1, removed)
	assert.NoFileExists(t, cacheFile)
}

func TestCleanCacheCommandReportsLockedEntry(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows keeps the process working directory locked")
	}

	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	locked := filepath.Join(home, "downloads", "locked")
	require.NoError(t, os.MkdirAll(locked, 0o755))

	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(locked))
	t.Cleanup(func() {
		_ = os.Chdir(wd)
		_ = os.RemoveAll(filepath.Join(home, "downloads"))
	})

	err = cleanCacheCmd.RunE(cleanCacheCmd, nil)

	require.Error(t, err)
	assert.DirExists(t, locked)
}
