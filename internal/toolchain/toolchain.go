package toolchain

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/target"
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

	if name.Channel != UnknownChannel {
		if IsVersionSelector(name.Version) {
			return findInstalledSelector(name)
		}
		dir, err := FindInstalledByName(name.String())
		if !errors.Is(err, os.ErrNotExist) || name.Target != "" {
			return dir, err
		}
		_, settings, err := config.LoadDefaultSettings()
		if err != nil {
			return "", err
		}
		host, err := target.CurrentHostTuple(settings.DefaultHost)
		if err != nil {
			return "", err
		}
		alias := name
		if name.Host != "" {
			alias.Host = ""
		} else {
			alias.Host = host
		}
		dir, err = FindInstalledByName(alias.String())
		if err != nil {
			return "", err
		}
		record, err := ReadInstallation(dir)
		if err != nil {
			return "", err
		}
		tuple := record.Tuple
		if tuple == "" {
			tuple = host
		}
		if name.Host != "" && name.Host != tuple || name.Host == "" && host != tuple {
			return "", os.ErrNotExist
		}
		return dir, nil
	}

	// For bare version numbers (UnknownChannel), search across all channels
	if name.Version != "" {
		if IsVersionSelector(name.Version) {
			return findInstalledSelector(name)
		}
		for _, ch := range []Channel{LTS, STS, Nightly} {
			candidateName := name
			candidateName.Channel = ch
			candidate, err := FindInstalled(candidateName)
			if err == nil {
				return candidate, nil
			}
			if !errors.Is(err, os.ErrNotExist) {
				return "", err
			}
		}
	}

	return "", os.ErrNotExist
}

// CanonicalHostName preserves the legacy unsuffixed directory and gives an
// explicit spelling of that same platform a single owner for cross SDKs.
func CanonicalHostName(name ToolchainName, host string) (ToolchainName, error) {
	if name.IsCustom() || name.Target != "" || name.Channel == UnknownChannel {
		return name, nil
	}
	if dir, err := FindInstalled(name); err == nil {
		actual, err := ParseToolchainName(filepath.Base(dir))
		if err != nil {
			return name, err
		}
		name.Host = actual.Host
		return name, nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return name, err
	}
	if name.Host == host {
		plain := name
		plain.Host = ""
		if _, err := FindInstalledByName(plain.String()); errors.Is(err, os.ErrNotExist) {
			return plain, nil
		} else if err != nil {
			return name, err
		}
	}
	return name, nil
}

func findInstalledSelector(name ToolchainName) (string, error) {
	names, err := ListInstalled()
	if err != nil {
		return "", err
	}
	var best ToolchainName
	for _, identity := range names {
		candidate, err := ParseToolchainName(identity)
		if err != nil || candidate.IsCustom() || candidate.Target != name.Target || name.Channel != UnknownChannel && candidate.Channel != name.Channel || !MatchesVersionSelector(name.Version, candidate.Version) {
			continue
		}
		if candidate.Host != name.Host {
			_, settings, err := config.LoadDefaultSettings()
			if err != nil {
				return "", err
			}
			wanted := name.Host
			if wanted == "" {
				wanted, err = target.CurrentHostTuple(settings.DefaultHost)
				if err != nil {
					return "", err
				}
			}
			dir, err := FindInstalledByName(identity)
			if err != nil {
				return "", err
			}
			record, err := ReadInstallation(dir)
			if err != nil {
				return "", err
			}
			actual := record.Tuple
			if actual == "" {
				actual, err = target.CurrentHostTuple(settings.DefaultHost)
				if err != nil {
					return "", err
				}
			}
			if actual != wanted {
				continue
			}
		}
		if best.Version == "" || compareSemVer(candidate.Version, best.Version) > 0 {
			best = candidate
		}
	}
	if best.Version == "" {
		return "", os.ErrNotExist
	}
	return FindInstalledByName(best.String())
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
