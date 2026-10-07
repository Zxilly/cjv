package component

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// checkEditRoots rejects links at the owned roots and metadata directories.
// Links below a component root remain valid ownership entries: removing a
// tracked leaf link unlinks it without touching the user's source directory.
func checkEditRoots(roots Roots, names []Name) error {
	if err := checkOwnedDirectories(roots.TcDir, metaPath(roots.TcDir)); err != nil {
		return err
	}
	for _, name := range names {
		spec, err := SpecFor(name)
		if err != nil {
			return err
		}
		anchor := roots.StdxDir
		if spec.Location.Anchor == AnchorDocs {
			anchor = roots.DocsDir
		}
		if err := checkOwnedDirectories(anchor, spec.InstallRoot(roots)); err != nil {
			return err
		}
	}
	return nil
}

func checkOwnedDirectories(anchor, path string) error {
	anchor, path = filepath.Clean(anchor), filepath.Clean(path)
	for {
		info, err := os.Lstat(path)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		if err == nil && info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("component root must not be a symlink: %s", path)
		}
		if path == anchor {
			return nil
		}
		parent := filepath.Dir(path)
		if parent == path {
			return fmt.Errorf("component path %s escapes its root %s", path, anchor)
		}
		path = parent
	}
}

// removalPaths validates the complete ownership set before the first delete.
// Keeping this plan in the editing module makes remove, link replacement and
// both archive paths obey the same rules, including shared-file ownership.
func removalPaths(roots Roots, name Name) ([]string, error) {
	if err := checkEditRoots(roots, []Name{name}); err != nil {
		return nil, err
	}
	paths, err := ReadManifest(roots.TcDir, name)
	if err != nil {
		return nil, err
	}
	spec, err := SpecFor(name)
	if err != nil {
		return nil, err
	}
	root := filepath.Clean(spec.InstallRoot(roots))
	for _, path := range paths {
		local := filepath.FromSlash(path)
		if !filepath.IsLocal(local) || filepath.Clean(local) == "." {
			return nil, fmt.Errorf("component manifest path escapes its root: %s", path)
		}
		for parent := filepath.Dir(filepath.Join(root, local)); parent != root; parent = filepath.Dir(parent) {
			info, err := os.Lstat(parent)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return nil, err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return nil, fmt.Errorf("component replacement would traverse manifest parent symlink: %s", parent)
			}
		}
	}
	keep, err := otherComponentClaims(roots, name)
	if err != nil {
		return nil, err
	}
	var removed []string
	for _, path := range paths {
		if !keep[filepath.Clean(filepath.FromSlash(path))] {
			removed = append(removed, path)
		}
	}
	return removed, nil
}

// checkMerge validates every incoming directory before modifying live files.
// A tracked leaf link scheduled for removal is safe: replacement will unlink
// it first. Untracked or shared links must never become merge destinations.
func checkMerge(source, dest string, removed []string) error {
	removedPaths := make(map[string]bool, len(removed))
	for _, path := range removed {
		removedPaths[filepath.Clean(filepath.FromSlash(path))] = true
	}
	return filepath.WalkDir(source, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		if removedPaths[rel] {
			return filepath.SkipDir
		}
		destination := filepath.Join(dest, rel)
		info, err := os.Lstat(destination)
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("component replacement would traverse untracked symlink: %s", destination)
		}
		return nil
	})
}
