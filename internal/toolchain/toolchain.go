package toolchain

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Zxilly/cjv/internal/config"
	goversion "github.com/hashicorp/go-version"
)

// compareSemVer compares two version strings (e.g. "1.0.5", "1.10.0").
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
func compareSemVer(a, b string) int {
	va, aErr := goversion.NewVersion(a)
	vb, bErr := goversion.NewVersion(b)
	switch {
	case aErr == nil && bErr == nil:
		return va.Compare(vb)
	case aErr == nil:
		return 1
	case bErr == nil:
		return -1
	default:
		return strings.Compare(a, b)
	}
}

func FindInstalled(name ToolchainName) (string, error) {
	// Custom/linked toolchains are looked up by exact directory name
	if name.IsCustom() {
		return FindInstalledByName(name.Custom)
	}

	tcDir, err := config.ToolchainsDir()
	if err != nil {
		return "", err
	}

	if name.Channel != UnknownChannel {
		return FindInstalledByName(name.String())
	}

	// For bare version numbers (UnknownChannel), search across all channels
	if name.Version != "" {
		for _, ch := range []Channel{LTS, STS, Nightly} {
			candidate := filepath.Join(tcDir, ch.String()+"-"+name.Version)
			if _, err := os.Stat(candidate); err == nil {
				return candidate, nil
			}
		}
	}

	return "", os.ErrNotExist
}

// FindInstalledByName looks up a toolchain by its exact directory name
// (e.g. a custom-linked toolchain like "my-sdk").
func FindInstalledByName(name string) (string, error) {
	dir, err := config.ToolchainDirFor(name)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(dir); err != nil {
		return "", err
	}
	return dir, nil
}

func ListInstalled() ([]string, error) {
	tcDir, err := config.ToolchainsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(tcDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var names []string
	for _, e := range entries {
		if !e.IsDir() && e.Type()&os.ModeSymlink == 0 {
			continue
		}
		name := e.Name()
		// Skip staging and backup directories
		if config.IsScratchName(name) {
			continue
		}
		names = append(names, name)
	}
	slices.Sort(names)
	return names, nil
}
