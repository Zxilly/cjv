//go:build !windows

package selfcmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/i18n"
)

// removeHomeDir removes the cjv home directory.
func removeHomeDir(home, managedExe string) error {
	// Safety: refuse to delete dangerous paths (filesystem root, user home, etc.)
	cleaned := filepath.Clean(home)
	if cleaned == "/" || cleaned == "." {
		return fmt.Errorf(i18n.T("SelfUnsafePath", nil), home)
	}
	if userHome, err := os.UserHomeDir(); err == nil && filepath.Clean(userHome) == cleaned {
		return fmt.Errorf(i18n.T("SelfUnsafePath", nil), home)
	}
	// Verify this is actually a cjv home directory.
	if _, err := os.Stat(managedExe); err != nil {
		return fmt.Errorf(i18n.T("SelfNotHome", nil), home)
	}

	if err := os.RemoveAll(home); err != nil {
		return fmt.Errorf(i18n.T("SelfRemoveFailed", nil), home, err)
	}
	return nil
}
