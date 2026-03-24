//go:build windows

package selfmgmt

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/sys/windows"
)

var currentExecutablePath = os.Executable

var startDetachedUninstallCleanup = func(home string) error {
	script := `Start-Sleep -Seconds 3; Remove-Item -LiteralPath $env:CJV_UNINSTALL_DIR -Recurse -Force`
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Env = append(os.Environ(), "CJV_UNINSTALL_DIR="+home)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP,
	}
	return cmd.Start()
}

// removeHomeDir removes the cjv home directory on Windows.
//
// If the managed cjv binary is the one currently running, a detached cleanup
// process deletes the directory after this process exits. Otherwise the managed
// home can be removed immediately.
func removeHomeDir(home, managedExe string) error {
	cleaned := filepath.Clean(home)
	if cleaned == "/" || cleaned == "\\" || cleaned == "." ||
		(len(cleaned) == 3 && cleaned[1] == ':' && (cleaned[2] == '/' || cleaned[2] == '\\')) {
		return fmt.Errorf("refusing to remove dangerous path: %s", home)
	}
	if userHome, err := os.UserHomeDir(); err == nil && filepath.Clean(userHome) == cleaned {
		return fmt.Errorf("refusing to remove dangerous path: %s", home)
	}
	if _, err := os.Stat(managedExe); err != nil {
		return fmt.Errorf("path %s does not appear to be a cjv home directory (managed binary not found)", home)
	}

	currentExe, err := currentExecutablePath()
	if err != nil {
		return err
	}
	if !strings.EqualFold(filepath.Clean(currentExe), filepath.Clean(managedExe)) {
		if err := os.RemoveAll(home); err != nil {
			return fmt.Errorf("failed to remove %s: %w", home, err)
		}
		return nil
	}

	// Remove symlinks first — Remove-Item -Recurse can fail on them.
	if err := removeSymlinks(filepath.Join(home, "bin")); err != nil {
		return fmt.Errorf("failed to remove symlinks in bin directory: %w", err)
	}

	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return err
	}
	ext := filepath.Ext(managedExe)
	stem := strings.TrimSuffix(filepath.Base(managedExe), ext)
	gcName := filepath.Join(filepath.Dir(managedExe), stem+"-gc-"+hex.EncodeToString(b)+ext)
	if err := os.Rename(managedExe, gcName); err != nil {
		return fmt.Errorf("failed to prepare managed binary for removal: %w", err)
	}

	if err := startDetachedUninstallCleanup(home); err != nil {
		if restoreErr := os.Rename(gcName, managedExe); restoreErr != nil {
			return fmt.Errorf("failed to start detached uninstall cleanup: %w; additionally failed to restore managed binary: %w", err, restoreErr)
		}
		return fmt.Errorf("failed to start detached uninstall cleanup: %w", err)
	}

	return nil
}

func removeSymlinks(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		if e.Type()&os.ModeSymlink != 0 {
			if err := os.Remove(filepath.Join(dir, e.Name())); err != nil {
				return err
			}
		}
	}
	return nil
}
