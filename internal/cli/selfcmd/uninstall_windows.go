//go:build windows

package selfcmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"

	"github.com/Zxilly/cjv/internal/i18n"
	"golang.org/x/sys/windows"
)

var currentExecutablePath = os.Executable

var startDetachedUninstallCleanup = func(home string) error {
	cmd := uninstallCleanupCommand(home, os.Getpid())
	if err := cmd.Start(); err != nil {
		return err
	}
	_ = cmd.Process.Release() //nolint:errcheck // the detached helper owns its lifetime
	return nil
}

func uninstallCleanupCommand(home string, parentPID int) *exec.Cmd {
	script := `Wait-Process -Id $env:CJV_UNINSTALL_PID -ErrorAction SilentlyContinue
Get-ChildItem -LiteralPath (Join-Path $env:CJV_UNINSTALL_DIR 'home\bin') -Force -ErrorAction SilentlyContinue | Where-Object { $_.LinkType -eq 'SymbolicLink' } | ForEach-Object { Remove-Item -LiteralPath $_.FullName -Force }
Remove-Item -LiteralPath $env:CJV_UNINSTALL_DIR -Recurse -Force`
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Env = append(os.Environ(), "CJV_UNINSTALL_DIR="+home, "CJV_UNINSTALL_PID="+strconv.Itoa(parentPID))
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP,
		HideWindow:    true,
	}
	return cmd
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
		return fmt.Errorf(i18n.T("SelfUnsafePath", nil), home)
	}
	if userHome, err := os.UserHomeDir(); err == nil && filepath.Clean(userHome) == cleaned {
		return fmt.Errorf(i18n.T("SelfUnsafePath", nil), home)
	}
	if _, err := os.Stat(managedExe); err != nil {
		return fmt.Errorf(i18n.T("SelfNotHome", nil), home)
	}

	currentExe, err := currentExecutablePath()
	if err != nil {
		return err
	}
	if !strings.EqualFold(filepath.Clean(currentExe), filepath.Clean(managedExe)) {
		if err := os.RemoveAll(home); err != nil {
			return fmt.Errorf(i18n.T("SelfRemoveFailed", nil), home, err)
		}
		return nil
	}

	// Retire this installation under a unique sibling before scheduling any
	// deletion. A new installation can immediately reuse home without becoming
	// the target of this operation's delayed cleanup.
	retired, err := os.MkdirTemp(filepath.Dir(cleaned), ".cjv-uninstall-*")
	if err != nil {
		return err
	}
	retiredHome := filepath.Join(retired, "home")
	if err := os.Rename(cleaned, retiredHome); err != nil {
		_ = os.Remove(retired) //nolint:errcheck // empty reservation only
		return fmt.Errorf(i18n.T("SelfPrepareRemovalFailed", nil), err)
	}

	if err := startDetachedUninstallCleanup(retired); err != nil {
		if restoreErr := os.Rename(retiredHome, cleaned); restoreErr != nil {
			return fmt.Errorf(i18n.T("SelfCleanupRestoreFailed", nil), err, restoreErr)
		}
		_ = os.Remove(retired) //nolint:errcheck // empty reservation only
		return fmt.Errorf(i18n.T("SelfCleanupFailed", nil), err)
	}

	return nil
}
