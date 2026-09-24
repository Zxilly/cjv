//go:build !windows

package fsops

func isWindowsSharingViolation(_ error) bool {
	return false
}

func isWindowsDirNotEmpty(_ error) bool {
	return false
}
