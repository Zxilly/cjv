//go:build windows

package utils

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for Windows retry classification -- controls whether file
// operations retry on transient locks from virus scanners.

func TestIsRetryableError_PermissionIsRetryableOnWindows(t *testing.T) {
	// On Windows, permission errors from antivirus are transient.
	assert.True(t, IsRetryableError(os.ErrPermission))
}

func TestIsRetryableError_SharingViolation(t *testing.T) {
	// ERROR_SHARING_VIOLATION (32) — file is locked by another process.
	err := syscall.Errno(32)
	assert.True(t, IsRetryableError(err))
}

func TestIsRetryableError_DirNotEmpty(t *testing.T) {
	// ERROR_DIR_NOT_EMPTY (145) — antivirus briefly holds directory handle.
	err := syscall.Errno(145)
	assert.True(t, IsRetryableError(err))
}

func TestIsRetryableError_WrappedErrors(t *testing.T) {
	// Errors are often wrapped; classification must work through wrapping.
	wrapped := fmt.Errorf("remove failed: %w", syscall.Errno(32))
	assert.True(t, IsRetryableError(wrapped))
}

func TestIsRetryableError_RegularErrorsAreNotRetryable(t *testing.T) {
	assert.False(t, IsRetryableError(errors.New("file not found")))
}

func TestIsWindowsSharingViolation(t *testing.T) {
	assert.True(t, isWindowsSharingViolation(syscall.Errno(32)))
	assert.False(t, isWindowsSharingViolation(syscall.Errno(5)))
	assert.False(t, isWindowsSharingViolation(errors.New("not errno")))
}

func TestIsWindowsDirNotEmpty(t *testing.T) {
	assert.True(t, isWindowsDirNotEmpty(syscall.Errno(145)))
	assert.False(t, isWindowsDirNotEmpty(syscall.Errno(5)))
}

// Tests for RemoveAllRetry -- used during uninstall to remove SDK
// directories. Must handle transient antivirus locks.

func TestRemoveAllRetry_RemovesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sdk")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "bin"), []byte("x"), 0o644))

	require.NoError(t, RemoveAllRetry(dir))

	_, err := os.Stat(dir)
	assert.True(t, os.IsNotExist(err))
}

func TestRemoveAllRetry_NonExistentIsNotError(t *testing.T) {
	err := RemoveAllRetry(filepath.Join(t.TempDir(), "nonexistent"))
	assert.NoError(t, err)
}
