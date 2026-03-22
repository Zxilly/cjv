//go:build !windows

package selfmgmt

import (
	"log/slog"
	"os"
	"os/user"
	"syscall"
)

// CheckSudoSafety warns if running under sudo with mismatched HOME,
// which could install toolchains to the wrong home directory.
func CheckSudoSafety() {
	// Only check if SUDO_USER is set (running under sudo)
	sudoUser := os.Getenv("SUDO_USER")
	if sudoUser == "" {
		return
	}

	// Check if effective UID is root
	if syscall.Geteuid() != 0 {
		return
	}

	// Compare HOME with the sudo user's actual home
	sudoUID := os.Getenv("SUDO_UID")
	if sudoUID == "" {
		return
	}

	u, err := user.LookupId(sudoUID)
	if err != nil {
		return
	}

	home := os.Getenv("HOME")
	if home != u.HomeDir {
		slog.Warn("running cjv under sudo may install toolchains to root's home directory", "user", sudoUser)
	}
}
