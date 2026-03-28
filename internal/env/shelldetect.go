package env

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Zxilly/cjv/internal/utils"
)

// ClassifyShellName maps a process name to a ShellType.
// Returns (shell, true) on match, (ShellPosix, false) on unknown.
func ClassifyShellName(name string) (ShellType, bool) {
	base := filepath.Base(name)
	if runtime.GOOS == "windows" {
		base = strings.TrimSuffix(strings.ToLower(base), ".exe")
	}

	switch base {
	case "bash", "zsh", "sh", "dash", "ksh":
		return ShellPosix, true
	case "fish":
		return ShellFish, true
	case "powershell", "pwsh":
		return ShellPowerShell, true
	case "cmd":
		return ShellCmd, true
	default:
		return ShellPosix, false
	}
}

// DetectShell attempts to detect the shell type from the parent process.
// Falls back to DefaultShellType() if detection fails.
func DetectShell() (ShellType, bool) {
	ppid := os.Getppid()
	name, err := utils.ProcessName(ppid)
	if err != nil {
		return DefaultShellType(), false
	}
	return ClassifyShellName(name)
}
