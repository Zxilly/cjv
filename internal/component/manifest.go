package component

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Zxilly/cjv/internal/utils"
)

// MetaDir holds component metadata under a hidden subdir to avoid colliding
// with SDK-managed directories.
const MetaDir = ".cjv/components"

const componentsFile = "components"
const manifestPrefix = "manifest-"

func metaPath(tcDir string, parts ...string) string {
	all := append([]string{tcDir, MetaDir}, parts...)
	return filepath.Join(all...)
}

func manifestPath(tcDir string, name Name) string {
	return metaPath(tcDir, manifestPrefix+string(name))
}

func IsInstalled(tcDir string, name Name) bool {
	_, err := os.Stat(manifestPath(tcDir, name))
	return err == nil
}

// ListInstalled silently drops index entries whose manifest is missing so a
// half-finished install / remove never poisons later commands.
func ListInstalled(tcDir string) ([]Name, error) {
	data, err := os.ReadFile(metaPath(tcDir, componentsFile))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []Name
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		n := Name(line)
		if _, ok := specs[n]; !ok {
			continue
		}
		if !IsInstalled(tcDir, n) {
			continue
		}
		out = append(out, n)
	}
	slices.Sort(out)
	return out, nil
}

// ReadManifest returns paths in forward-slash form, rooted at spec.InstallRoot.
func ReadManifest(tcDir string, name Name) ([]string, error) {
	data, err := os.ReadFile(manifestPath(tcDir, name))
	if err != nil {
		return nil, err
	}
	var out []string
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, line)
	}
	return out, nil
}

// WriteManifest expects relPaths in forward-slash form, rooted at spec.InstallRoot.
func WriteManifest(tcDir string, name Name, relPaths []string) error {
	dir := metaPath(tcDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	body := strings.Join(relPaths, "\n")
	if body != "" {
		body += "\n"
	}
	if err := utils.WriteFileAtomic(filepath.Join(dir, manifestPrefix+string(name)), []byte(body), 0o644); err != nil {
		return err
	}
	return addToComponentsIndex(tcDir, name)
}

func addToComponentsIndex(tcDir string, name Name) error {
	current, err := readComponentsIndex(tcDir)
	if err != nil {
		return err
	}
	if slices.Contains(current, name) {
		return nil
	}
	current = append(current, name)
	slices.Sort(current)
	return writeComponentsIndex(tcDir, current)
}

func removeFromComponentsIndex(tcDir string, name Name) error {
	current, err := readComponentsIndex(tcDir)
	if err != nil {
		return err
	}
	idx := slices.Index(current, name)
	if idx < 0 {
		return nil
	}
	current = slices.Delete(current, idx, idx+1)
	return writeComponentsIndex(tcDir, current)
}

func readComponentsIndex(tcDir string) ([]Name, error) {
	data, err := os.ReadFile(metaPath(tcDir, componentsFile))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	var out []Name
	for line := range strings.SplitSeq(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		out = append(out, Name(line))
	}
	return out, nil
}

func writeComponentsIndex(tcDir string, names []Name) error {
	dir := metaPath(tcDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	var b strings.Builder
	for _, n := range names {
		b.WriteString(string(n))
		b.WriteByte('\n')
	}
	return utils.WriteFileAtomic(filepath.Join(dir, componentsFile), []byte(b.String()), 0o644)
}

// Remove drops a component's manifest and the files it tracks. Files also
// claimed by another installed component (mdBook archives ship overlapping
// static assets) stay on disk.
func Remove(roots Roots, name Name) error {
	myPaths, err := ReadManifest(roots.TcDir, name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// Manifest missing — best-effort tidy of the index.
			return removeFromComponentsIndex(roots.TcDir, name)
		}
		return err
	}
	if err := removePaths(roots, name, myPaths); err != nil {
		return err
	}
	if err := os.Remove(manifestPath(roots.TcDir, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return removeFromComponentsIndex(roots.TcDir, name)
}

// removePaths skips paths still claimed by another component, then prunes
// the now-empty directories up to (and including) the install root.
func removePaths(roots Roots, name Name, paths []string) error {
	spec, err := SpecFor(name)
	if err != nil {
		return err
	}
	installRoot := spec.InstallRoot(roots)
	keep := otherComponentClaims(roots, name)

	for _, rel := range paths {
		if keep[rel] {
			continue
		}
		abs := filepath.Join(installRoot, filepath.FromSlash(rel))
		if err := os.Remove(abs); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove %s: %w", abs, err)
		}
	}

	rootStop := filepath.Clean(installRoot)
	pruneSet := make(map[string]struct{})
	for _, rel := range paths {
		if keep[rel] {
			continue
		}
		dir := filepath.Dir(filepath.Join(installRoot, filepath.FromSlash(rel)))
		for {
			clean := filepath.Clean(dir)
			if clean == rootStop {
				break
			}
			pruneSet[clean] = struct{}{}
			parent := filepath.Dir(clean)
			if parent == clean {
				break
			}
			dir = parent
		}
	}
	prune := make([]string, 0, len(pruneSet))
	for p := range pruneSet {
		prune = append(prune, p)
	}
	slices.SortFunc(prune, func(a, b string) int {
		return strings.Count(b, string(filepath.Separator)) - strings.Count(a, string(filepath.Separator))
	})
	for _, dir := range prune {
		_ = os.Remove(dir) //nolint:errcheck // best-effort: non-empty dirs stay
	}
	_ = os.Remove(rootStop) //nolint:errcheck // best-effort
	return nil
}

// otherComponentClaims returns the set of relative paths that belong to
// installed components other than `excluding` AND share the same install
// root. Used by Remove to keep shared static assets alive when overlapping
// components remain.
func otherComponentClaims(roots Roots, excluding Name) map[string]bool {
	claims := make(map[string]bool)
	excludingSpec, err := SpecFor(excluding)
	if err != nil {
		return claims
	}
	installed, _ := ListInstalled(roots.TcDir)
	if len(installed) == 0 || (len(installed) == 1 && installed[0] == excluding) {
		return claims
	}
	excludingRoot := excludingSpec.InstallRoot(roots)
	for _, n := range installed {
		if n == excluding {
			continue
		}
		other, err := SpecFor(n)
		if err != nil || other.InstallRoot(roots) != excludingRoot {
			continue
		}
		paths, err := ReadManifest(roots.TcDir, n)
		if err != nil {
			continue
		}
		for _, p := range paths {
			claims[p] = true
		}
	}
	return claims
}
