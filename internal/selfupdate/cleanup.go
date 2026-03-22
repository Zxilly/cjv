package selfupdate

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// CleanupOldBinaries removes stale updater/uninstall leftovers from the managed
// cjv bin directory.
func CleanupOldBinaries() {
	if runtime.GOOS != "windows" {
		return
	}

	managedExe, err := ManagedExecutablePath()
	if err != nil {
		return
	}

	dir := filepath.Dir(managedExe)
	base := filepath.Base(managedExe)
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}

	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, stem+"-gc-") && strings.HasSuffix(name, filepath.Ext(base)) {
			_ = os.Remove(filepath.Join(dir, name))
			continue
		}
		if name == "."+base+".old" || (strings.HasPrefix(name, base) && strings.HasSuffix(name, ".old")) {
			_ = os.Remove(filepath.Join(dir, name))
		}
	}
}
