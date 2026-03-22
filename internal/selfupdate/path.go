package selfupdate

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/Zxilly/cjv/internal/utils"
)

// ManagedExecutablePath returns the cjv binary managed under CJV_HOME/bin.
// It is anchored to the installed bin directory rather than the currently
// running executable path, so update/uninstall always target the managed copy.
func ManagedExecutablePath() (string, error) {
	binDir, err := config.BinDir()
	if err != nil {
		return "", err
	}

	managed := filepath.Join(binDir, proxy.CjvBinaryName())
	if _, err := os.Stat(managed); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", fmt.Errorf("managed cjv binary not found at %s", managed)
		}
		return "", fmt.Errorf("failed to access managed cjv binary %s: %w", managed, err)
	}

	return managed, nil
}

// EnsureManagedExecutable bootstraps the managed cjv binary under CJV_HOME/bin
// from the currently running executable when it is missing.
func EnsureManagedExecutable() (string, error) {
	binDir, err := config.BinDir()
	if err != nil {
		return "", err
	}

	managed := filepath.Join(binDir, proxy.CjvBinaryName())
	if _, err := os.Stat(managed); err == nil {
		return managed, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("failed to access managed cjv binary %s: %w", managed, err)
	}

	currentExe, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("failed to locate current cjv executable: %w", err)
	}
	info, err := os.Stat(currentExe)
	if err != nil {
		return "", fmt.Errorf("failed to stat current cjv executable %s: %w", currentExe, err)
	}
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return "", err
	}
	if err := utils.CopyFile(currentExe, managed, info.Mode().Perm()); err != nil {
		return "", fmt.Errorf("failed to bootstrap managed cjv binary %s: %w", managed, err)
	}
	return managed, nil
}
