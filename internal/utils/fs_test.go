package utils

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateLink(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "source")
	require.NoError(t, os.WriteFile(src, []byte("binary"), 0o755))

	dst := filepath.Join(tmp, "link")
	require.NoError(t, CreateLink(src, dst))
	assert.FileExists(t, dst)
}

func TestCreateLinkPreservesExistingDestinationWhenReplacementFails(t *testing.T) {
	oldSymlink := createSymlink
	oldHardLink := createHardLink
	oldCopyFile := copyFileForLink
	defer func() {
		createSymlink = oldSymlink
		createHardLink = oldHardLink
		copyFileForLink = oldCopyFile
	}()

	createSymlink = func(string, string) error {
		return errors.New("symlink disabled")
	}
	createHardLink = func(string, string) error {
		return errors.New("hard link disabled")
	}
	copyFileForLink = func(string, string, os.FileMode) error {
		return errors.New("copy failed")
	}

	tmp := t.TempDir()
	src := filepath.Join(tmp, "source")
	dst := filepath.Join(tmp, "tool.exe")
	require.NoError(t, os.WriteFile(src, []byte("new"), 0o755))
	require.NoError(t, os.WriteFile(dst, []byte("old"), 0o755))

	err := CreateLink(src, dst)
	require.Error(t, err)

	data, readErr := os.ReadFile(dst)
	require.NoError(t, readErr)
	assert.Equal(t, []byte("old"), data)
}

// --- Tests merged from copy_file_test.go ---

// Tests for CopyFile -- used as a last-resort fallback when symlinks
// and hard links both fail (e.g., across filesystems on Windows).

func TestCopyFile_PreservesContent(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "original.bin")
	dst := filepath.Join(dir, "copy.bin")

	data := []byte("#!/usr/bin/env cjc\nprint(\"hello\")\n")
	require.NoError(t, os.WriteFile(src, data, 0o644))

	require.NoError(t, CopyFile(src, dst, 0o755))

	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, data, got, "copied file content must match source exactly")
}

func TestCopyFile_FailsOnMissingSource(t *testing.T) {
	dir := t.TempDir()
	err := CopyFile(filepath.Join(dir, "nonexistent"), filepath.Join(dir, "dst"), 0o644)
	assert.Error(t, err, "should fail when source file does not exist")
}

func TestCopyFile_LargeFile(t *testing.T) {
	// SDK binaries can be tens of megabytes; verify copy works for non-trivial sizes.
	dir := t.TempDir()
	src := filepath.Join(dir, "large.bin")
	dst := filepath.Join(dir, "large_copy.bin")

	data := make([]byte, 1<<16) // 64 KB
	for i := range data {
		data[i] = byte(i % 251) // prime modulus to avoid patterns
	}
	require.NoError(t, os.WriteFile(src, data, 0o644))

	require.NoError(t, CopyFile(src, dst, 0o755))

	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, data, got)
}
