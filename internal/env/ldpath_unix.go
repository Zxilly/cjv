//go:build !windows

package env

import (
	"os"
	"runtime"
	"strings"
)

// EnsureLibraryPath prepends SDK library directories to the appropriate
// library search path environment variable. It merges with the current
// process environment value to avoid overwriting user settings.
func EnsureLibraryPath(cfg *EnvConfig) {
	entries := libraryPathEntries(cfg)
	if len(entries) == 0 {
		return
	}

	ldKey := libraryPathKey(runtime.GOOS)
	if ldKey == "" {
		return
	}

	parts := append([]string(nil), entries...)
	if derivedVal := cfg.Vars[ldKey]; derivedVal != "" {
		parts = append(parts, derivedVal)
	}
	if processVal := os.Getenv(ldKey); processVal != "" {
		parts = append(parts, processVal)
	}

	seen := make(map[string]bool)
	var deduped []string
	for _, p := range parts {
		for entry := range strings.SplitSeq(p, string(os.PathListSeparator)) {
			if entry == "" {
				continue
			}
			key := canonicalEnvKey(entry)
			if seen[key] {
				continue
			}
			seen[key] = true
			deduped = append(deduped, entry)
		}
	}

	cfg.Vars[ldKey] = strings.Join(deduped, string(os.PathListSeparator))
}
