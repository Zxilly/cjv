//go:build !windows

package env

// AddPathToWindowsRegistry is a no-op on non-Windows platforms.
func AddPathToWindowsRegistry(binDir string) error {
	return nil
}

// RemovePathFromWindowsRegistry is a no-op on non-Windows platforms.
func RemovePathFromWindowsRegistry(binDir string) error {
	return nil
}
