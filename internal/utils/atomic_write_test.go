package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for WriteFileAtomic — used to save settings.toml and env.toml.
// If the process is interrupted during write, the file must be either
// old content or new content, never partially written (corrupted).

func TestWriteFileAtomic_CreatesNewFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.toml")
	data := []byte("[toolchain]\nchannel = \"lts\"\n")

	require.NoError(t, WriteFileAtomic(path, data, 0o644))

	got, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, data, got)
}

func TestWriteFileAtomic_OverwritesExistingFile(t *testing.T) {
	// When user changes settings, the old file must be fully replaced.
	dir := t.TempDir()
	path := filepath.Join(dir, "settings.toml")

	require.NoError(t, os.WriteFile(path, []byte("old content"), 0o644))
	require.NoError(t, WriteFileAtomic(path, []byte("new content"), 0o644))

	got, _ := os.ReadFile(path)
	assert.Equal(t, "new content", string(got))
}

func TestWriteFileAtomic_NoTempFileLeftBehind(t *testing.T) {
	// Temp files (.cjv-tmp-*) must be cleaned up after a successful write.
	// Leftover temp files waste disk space and confuse users.
	dir := t.TempDir()
	path := filepath.Join(dir, "data.bin")

	require.NoError(t, WriteFileAtomic(path, []byte("data"), 0o644))

	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		assert.False(t, strings.HasPrefix(e.Name(), ".cjv-tmp-"),
			"temp file %q should have been cleaned up", e.Name())
	}
}

func TestWriteFileAtomic_EmptyData(t *testing.T) {
	// Edge case: writing zero bytes should still create the file.
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.toml")

	require.NoError(t, WriteFileAtomic(path, []byte{}, 0o644))
	assert.FileExists(t, path)

	got, _ := os.ReadFile(path)
	assert.Empty(t, got)
}
