//go:build !windows

package env

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// EnsureLibraryPath prepends the SDK lib directory to the appropriate
// library search path environment variable. It merges with the current
// process environment value to avoid overwriting user settings.
func EnsureLibraryPath(cfg *EnvConfig, sdkDir string) {
	libDir := filepath.Join(sdkDir, "lib")
	if _, err := os.Stat(libDir); err != nil {
		return
	}

	var ldKey string
	switch runtime.GOOS {
	case "darwin":
		ldKey = "DYLD_FALLBACK_LIBRARY_PATH"
	default:
		ldKey = "LD_LIBRARY_PATH"
	}

	// Build the merged value: SDK lib + env.toml value + current process env
	parts := []string{libDir}

	if envTomlVal := cfg.Vars[ldKey]; envTomlVal != "" {
		parts = append(parts, envTomlVal)
	}

	if processVal := os.Getenv(ldKey); processVal != "" {
		parts = append(parts, processVal)
	} else if runtime.GOOS == "darwin" && ldKey == "DYLD_FALLBACK_LIBRARY_PATH" {
		// macOS: preserve system defaults when env var was unset
		home, _ := os.UserHomeDir()
		if home != "" {
			parts = append(parts, filepath.Join(home, "lib"))
		}
		parts = append(parts, "/usr/local/lib", "/usr/lib")
	}

	// Deduplicate
	seen := make(map[string]bool)
	var deduped []string
	for _, p := range parts {
		for _, entry := range strings.Split(p, string(os.PathListSeparator)) {
			if entry != "" && !seen[entry] {
				seen[entry] = true
				deduped = append(deduped, entry)
			}
		}
	}

	cfg.Vars[ldKey] = strings.Join(deduped, string(os.PathListSeparator))
}
