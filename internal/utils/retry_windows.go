//go:build windows

package utils

import (
	"errors"
	"syscall"
)

const (
	// errSharingViolation is ERROR_SHARING_VIOLATION: file is in use by another process.
	errSharingViolation syscall.Errno = 32
	// errDirNotEmpty is ERROR_DIR_NOT_EMPTY: transient when virus scanner holds a handle.
	errDirNotEmpty syscall.Errno = 145
)

func isWindowsErrno(err error, code syscall.Errno) bool {
	var errno syscall.Errno
	if errors.As(err, &errno) {
		return errno == code
	}
	return false
}

func isWindowsSharingViolation(err error) bool {
	return isWindowsErrno(err, errSharingViolation)
}

func isWindowsDirNotEmpty(err error) bool {
	return isWindowsErrno(err, errDirNotEmpty)
}
