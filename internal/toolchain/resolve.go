package toolchain

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
)

// ResolveActiveToolchain resolves the current active toolchain directory, name,
// and source. On error, tcName may still contain the configured (but uninstalled)
// toolchain name.
func ResolveActiveToolchain() (tcDir string, tcName string, source config.OverrideSource, err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to get working directory: %w", err)
	}
	sf, err := config.DefaultSettingsFile()
	if err != nil {
		return "", "", 0, err
	}
	settings, err := sf.Load()
	if err != nil {
		return "", "", 0, err
	}

	rawName, source, err := config.ResolveToolchain(settings, cwd)
	if err != nil {
		return "", "", 0, err
	}

	parsed, err := ParseToolchainName(rawName)
	if err != nil {
		return "", rawName, source, fmt.Errorf("invalid toolchain name '%s' (from %s): %w", rawName, source, err)
	}

	dir, findErr := FindInstalled(parsed)
	if findErr != nil {
		if !errors.Is(findErr, os.ErrNotExist) {
			return "", rawName, source, findErr
		}
		return "", rawName, source, &cjverr.ToolchainNotInstalledError{Name: rawName}
	}

	// Use the actual directory name as the display name to avoid
	// showing "unknown-X.Y.Z" for bare version inputs.
	return dir, filepath.Base(dir), source, nil
}
