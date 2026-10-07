package component

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/Zxilly/cjv/internal/fsops"
)

// StagePreparedBatch builds replacements in empty private staging roots. Only
// selected component roots and their metadata are copied, never the SDK. The
// caller keeps the installed roots stable and owns staging cleanup/publication.
// Applying normal ownership rules to these copies preserves untracked files,
// shared assets and user links without touching the live installation on error.
func StagePreparedBatch(roots, prepared, staging Roots, names []Name, force bool) error {
	if err := copyExisting(metaPath(roots.TcDir), metaPath(staging.TcDir)); err != nil {
		return err
	}
	var copied []string
	for _, name := range names {
		spec, err := SpecFor(name)
		if err != nil {
			return err
		}
		source := spec.InstallRoot(roots)
		if !slices.Contains(copied, source) {
			if err := copyExisting(source, spec.InstallRoot(staging)); err != nil {
				return err
			}
			copied = append(copied, source)
		}
	}
	for _, name := range names {
		if err := installPrepared(staging, prepared, name, force); err != nil {
			return err
		}
	}
	return nil
}

func copyExisting(source, dest string) error {
	info, err := os.Lstat(source)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	// A root link would make scratch mutations reach its external target.
	// Supported component links are children of the root and copy as links.
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("component root must not be a symlink: %s", source)
	}
	return fsops.CopyTree(source, dest)
}

// Manifest entries may have been edited, or their parent directories replaced
// by user links since installation. Removing the final link is safe; traversing
// a copied parent link would instead delete files outside private staging.
func checkPreparedRemoval(roots Roots, name Name) error {
	spec, err := SpecFor(name)
	if err != nil {
		return err
	}
	paths, err := ReadManifest(roots.TcDir, name)
	if err != nil {
		return err
	}
	root := filepath.Clean(spec.InstallRoot(roots))
	for _, path := range paths {
		local := filepath.FromSlash(path)
		if !filepath.IsLocal(local) || filepath.Clean(local) == "." {
			return fmt.Errorf("component manifest path escapes its root: %s", path)
		}
		for parent := filepath.Dir(filepath.Join(root, local)); parent != root; parent = filepath.Dir(parent) {
			info, err := os.Lstat(parent)
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			if err != nil {
				return err
			}
			if info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("component replacement would traverse manifest parent symlink: %s", parent)
			}
		}
	}
	return nil
}

// A copied untracked link still points outside scratch. Refuse to merge an
// incoming directory through it. Tracked links have already been removed by
// the ownership rules, and incoming leaf files replace links without following
// them. Walking directories first makes this check precede any merge writes.
func checkPreparedDirectories(source, dest string) error {
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
