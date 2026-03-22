//go:build !windows

package utils

func isWindowsSharingViolation(_ error) bool {
	return false
}

func isWindowsDirNotEmpty(_ error) bool {
	return false
}
