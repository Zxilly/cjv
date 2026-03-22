//go:build !windows

package selfmgmt

import (
	"fmt"
	"os"
	"path/filepath"
)

// removeHomeDir removes the cjv home directory.
func removeHomeDir(home, managedExe string) error {
	// Safety: refuse to delete dangerous paths (filesystem root, user home, etc.)
	cleaned := filepath.Clean(home)
	if cleaned == "/" || cleaned == "." {
		return fmt.Errorf("refusing to remove dangerous path: %s", home)
	}
	if userHome, err := os.UserHomeDir(); err == nil && filepath.Clean(userHome) == cleaned {
		return fmt.Errorf("refusing to remove dangerous path: %s", home)
	}
	// Verify this is actually a cjv home directory.
	if _, err := os.Stat(managedExe); err != nil {
		return fmt.Errorf("path %s does not appear to be a cjv home directory (managed binary not found)", home)
	}

	if err := os.RemoveAll(home); err != nil {
		return fmt.Errorf("failed to remove %s: %w", home, err)
	}
	return nil
}
