package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPurgeDownloadsDirRemovesAll(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	downloads := filepath.Join(home, "downloads")
	require.NoError(t, os.MkdirAll(downloads, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(downloads, "sdk-1.0.5.zip"), []byte("data"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(downloads, "sdk-2.0.0.tar.gz"), []byte("data"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(downloads, "partial"), 0o755))

	removed, err := purgeDownloadsDir()
	require.NoError(t, err)

	assert.Equal(t, 3, removed)

	_, err = os.Stat(filepath.Join(downloads, "sdk-1.0.5.zip"))
	assert.True(t, os.IsNotExist(err), "archive files should be removed")

	_, err = os.Stat(filepath.Join(downloads, "partial"))
	assert.True(t, os.IsNotExist(err), "subdirectories should also be removed")
}

func TestPurgeDownloadsDirEmptyOrMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	assert.NotPanics(t, func() {
		removed, err := purgeDownloadsDir()
		require.NoError(t, err)
		assert.Zero(t, removed)
	})
}

func TestPurgeDownloadsDirRemovesReadonlyEntry(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("readonly file deletion is Windows-specific")
	}

	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	cacheFile := filepath.Join(home, "downloads", "archive.zip")
	require.NoError(t, os.MkdirAll(filepath.Dir(cacheFile), 0o755))
	require.NoError(t, os.WriteFile(cacheFile, []byte("data"), 0o644))
	require.NoError(t, os.Chmod(cacheFile, 0o444))

	removed, err := purgeDownloadsDir()

	require.NoError(t, err)
	assert.Equal(t, 1, removed)
	assert.NoFileExists(t, cacheFile)
}
