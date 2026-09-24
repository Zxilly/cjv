// Package fsops is the filesystem layer every package that touches CJV_HOME
// goes through: removes and renames that survive the transient locks Windows
// virus scanners and indexers place on files, atomic file writes, directory
// links that fall back to junctions where symlinks need privileges, and the
// tree operations (MoveTree, CopyTree) that stage archives and back up
// component roots.
package fsops

import (
	"errors"
	"os"
	"runtime"

	"github.com/Zxilly/cjv/internal/retry"
)

// maxAttempts is how often a filesystem operation is retried on a transient error.
const maxAttempts = 10

// IsRetryableError returns true for transient filesystem errors
// (commonly caused by virus scanners or indexers on Windows).
// On Unix, permission errors are not transient and should not be retried.
func IsRetryableError(err error) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	// Windows: Permission denied, sharing violation, directory not empty
	// are commonly transient due to Defender, search indexer, etc.
	return errors.Is(err, os.ErrPermission) ||
		isWindowsDirNotEmpty(err) ||
		isWindowsSharingViolation(err)
}

// Retry runs fn, repeating it with backoff while it fails with a transient
// filesystem error (IsRetryableError). It is the retry policy behind
// RemoveAllRetry and RenameRetry, exposed for callers that operate through an
// os.Root and therefore cannot use the path-based helpers.
func Retry(fn func() error) error {
	return retry.Do(maxAttempts, IsRetryableError, fn)
}

// RemoveAllRetry removes a path with retry for transient errors.
func RemoveAllRetry(path string) error {
	return Retry(func() error {
		return os.RemoveAll(path)
	})
}

// RenameRetry renames with retry for transient errors.
func RenameRetry(oldpath, newpath string) error {
	return Retry(func() error {
		return os.Rename(oldpath, newpath)
	})
}
