package component

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/utils"
)

// Link points a component install root at a user-supplied local directory by
// creating per-child symlinks (junctions on Windows where unprivileged
// symlinks are disallowed). The manifest is written exactly like a normal
// install so that IsInstalled, ApplyEnv, component remove, and toolchain
// uninstall all keep working — removing manifest entries via os.Remove
// deletes the symlink, never the user's source directory. Returns the
// resolved absolute source path so callers can display it.
func Link(roots Roots, name Name, sourcePath string, force bool) (absSource string, retErr error) {
	spec, err := SpecFor(name)
	if err != nil {
		return "", err
	}
	if !spec.Linkable {
		return "", &cjverr.ComponentLinkNotSupportedError{Component: string(name)}
	}

	absSource, err = filepath.Abs(sourcePath)
	if err != nil {
		return "", &cjverr.ComponentLinkInvalidPathError{Reason: err.Error()}
	}
	if err := validateLinkSource(absSource, spec.LinkChildren); err != nil {
		return "", err
	}

	alreadyInstalled := IsInstalled(roots.TcDir, name)
	if !force && alreadyInstalled {
		return "", &cjverr.ComponentAlreadyInstalledError{
			Toolchain: filepath.Base(roots.TcDir),
			Component: string(name),
		}
	}

	var snap *Snapshot
	var created []string
	defer func() {
		if retErr == nil {
			return
		}
		_ = removePaths(roots, name, created)       //nolint:errcheck // best-effort rollback
		_ = cleanupComponentMeta(roots.TcDir, name) //nolint:errcheck // best-effort rollback
		if snap != nil {
			_ = snap.Restore() //nolint:errcheck // best-effort rollback
		}
	}()

	if force && alreadyInstalled {
		snap, err = TakeSnapshot(roots, []Name{name})
		if err != nil {
			return "", err
		}
		defer snap.Cleanup() //nolint:errcheck // best-effort cleanup
		if err := Remove(roots, name); err != nil {
			return "", fmt.Errorf("relink: remove existing %s: %w", name, err)
		}
	}

	destDir := spec.InstallRoot(roots)
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return "", err
	}

	for _, rel := range spec.LinkChildren {
		target := filepath.Join(absSource, rel)
		linkPath := filepath.Join(destDir, rel)
		// Mirror moveStagedFiles: clear any unmanaged leftover before placing
		// the new symlink, since SymlinkOrJunction won't overwrite.
		if _, lerr := os.Lstat(linkPath); lerr == nil {
			if rerr := os.RemoveAll(linkPath); rerr != nil {
				return "", fmt.Errorf("remove existing %s: %w", linkPath, rerr)
			}
		}
		if err := utils.SymlinkOrJunction(target, linkPath); err != nil {
			return "", fmt.Errorf("create link %s -> %s: %w", linkPath, target, err)
		}
		created = append(created, rel)
	}

	if err := WriteManifest(roots.TcDir, name, spec.LinkChildren); err != nil {
		return "", err
	}
	return absSource, nil
}

func validateLinkSource(absSource string, children []string) error {
	info, err := os.Stat(absSource)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &cjverr.ComponentLinkInvalidPathError{
				Reason: fmt.Sprintf("path does not exist: %s", absSource),
			}
		}
		return err
	}
	if !info.IsDir() {
		return &cjverr.ComponentLinkInvalidPathError{
			Reason: fmt.Sprintf("not a directory: %s", absSource),
		}
	}
	for _, sub := range children {
		subPath := filepath.Join(absSource, sub)
		subInfo, err := os.Stat(subPath)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return &cjverr.ComponentLinkInvalidPathError{
					Reason: fmt.Sprintf("missing required subdir %q under %s", sub, absSource),
				}
			}
			return err
		}
		if !subInfo.IsDir() {
			return &cjverr.ComponentLinkInvalidPathError{
				Reason: fmt.Sprintf("%q is not a directory under %s", sub, absSource),
			}
		}
	}
	return nil
}
