//go:build windows

package env

// EnsureLibraryPath is a no-op on Windows.
// Windows uses PATH for DLL discovery, which is already handled.
func EnsureLibraryPath(cfg *EnvConfig, sdkDir string) {}
