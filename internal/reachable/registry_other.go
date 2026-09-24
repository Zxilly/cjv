//go:build !windows

package reachable

// addPathToWindowsRegistry is a no-op on non-Windows platforms.
func addPathToWindowsRegistry(binDir string) error {
	return nil
}

// removePathFromWindowsRegistry is a no-op on non-Windows platforms.
func removePathFromWindowsRegistry(binDir string) error {
	return nil
}
