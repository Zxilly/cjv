package selfupdate

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/Zxilly/cjv/internal/utils"
)

// CleanupOldBinaries removes stale updater/uninstall leftovers from the managed
// cjv bin directory.
func CleanupOldBinaries() {
	if runtime.GOOS != "windows" {
		return
	}

	dir, err := config.BinDir()
	if err != nil {
		return
	}

	base := proxy.CjvBinaryName()
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	managedExe := filepath.Join(dir, base)
	managedExists := true
	if _, err := os.Stat(managedExe); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return
		}
		managedExists = false
	}

	var oldPaths []string
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, stem+"-gc-") && strings.HasSuffix(name, filepath.Ext(base)) {
			_ = os.Remove(filepath.Join(dir, name))
			continue
		}
		if name == "."+base+".old" || (strings.HasPrefix(name, base) && strings.HasSuffix(name, ".old")) {
			oldPaths = append(oldPaths, filepath.Join(dir, name))
		}
	}

	restored := ""
	if !managedExists {
		for _, oldPath := range oldPaths {
			if err := utils.RenameRetry(oldPath, managedExe); err == nil {
				restored = oldPath
				managedExists = true
				break
			} else {
				slog.Warn("failed to restore old managed binary", "old", oldPath, "managed", managedExe, "error", err)
			}
		}
	}
	if !managedExists {
		return
	}

	for _, oldPath := range oldPaths {
		if oldPath != restored {
			_ = os.Remove(oldPath)
		}
	}
}
