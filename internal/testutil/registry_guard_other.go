//go:build !windows

package testutil

// RegistryPathGuard is a no-op on non-Windows platforms.
type RegistryPathGuard struct{}

// SaveRegistryPath is a no-op on non-Windows platforms.
func SaveRegistryPath() (*RegistryPathGuard, error) {
	return &RegistryPathGuard{}, nil
}

// Restore is a no-op on non-Windows platforms.
func (g *RegistryPathGuard) Restore() {}

// ReadRegistryPath is a no-op on non-Windows platforms.
func ReadRegistryPath() (string, error) {
	return "", nil
}
