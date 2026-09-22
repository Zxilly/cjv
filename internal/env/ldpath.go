package env

import (
	"os"
	"path/filepath"
)

func libraryPathEntries(cfg *EnvConfig) []string {
	if cfg == nil {
		return nil
	}
	var entries []string
	seen := make(map[string]bool)
	for _, entry := range cfg.LibraryPathPrepend {
		if entry == "" {
			continue
		}
		key := canonicalEnvKey(filepath.Clean(entry))
		if seen[key] {
			continue
		}
		info, err := os.Stat(entry)
		if err != nil || !info.IsDir() {
			continue
		}
		seen[key] = true
		entries = append(entries, entry)
	}
	return entries
}

func libraryPathKey(goos string) string {
	switch goos {
	case "windows":
		return ""
	case "darwin":
		return "DYLD_LIBRARY_PATH"
	default:
		return "LD_LIBRARY_PATH"
	}
}
