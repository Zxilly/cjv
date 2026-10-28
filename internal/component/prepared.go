package component

import (
	"errors"
	"fmt"
	"os"
	"slices"

	"github.com/Zxilly/cjv/internal/fsops"
)

// StagePreparedBatch builds replacements in empty private staging roots. Only
// selected component roots and their metadata are copied, never the SDK. The
// caller keeps the installed roots stable and owns staging cleanup/publication.
// Applying normal ownership rules to these copies preserves untracked files,
// shared assets and user links without touching the live installation on error.
func StagePreparedBatch(roots, prepared, staging Roots, names []Name, force bool) error {
	return StagePreparedChanges(roots, prepared, staging, names, nil, force)
}

// StagePreparedChanges also handles components removed by a toolchain group
// plan. All affected roots and their metadata remain private until the caller
// publishes them together with any SDK or target changes.
func StagePreparedChanges(roots, prepared, staging Roots, added, removed []Name, force bool) error {
	names := append(slices.Clone(added), removed...)
	if err := checkEditRoots(roots, names); err != nil {
		return err
	}
	if err := checkEditRoots(staging, names); err != nil {
		return err
	}
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
	for _, name := range removed {
		if err := Remove(staging, name); err != nil {
			return err
		}
	}
	for _, name := range added {
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
