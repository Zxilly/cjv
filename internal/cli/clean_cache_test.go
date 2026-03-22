package cli

import (
	"os"
	"path/filepath"
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

	removed := cleanDownloadCache()

	// All entries should be removed
	assert.Equal(t, 3, removed)

	_, err := os.Stat(filepath.Join(downloads, "sdk-1.0.5.zip"))
	assert.True(t, os.IsNotExist(err), "archive files should be removed")

	_, err = os.Stat(filepath.Join(downloads, "partial"))
	assert.True(t, os.IsNotExist(err), "subdirectories should also be removed")
}

func TestCleanDownloadCache_EmptyOrMissingDir(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	// No downloads directory — should not panic
	assert.NotPanics(t, func() { cleanDownloadCache() })
}
