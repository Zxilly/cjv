package fsops

import (
	"fmt"
	"os"
	"path/filepath"
)

// MoveTree merges every entry under srcDir into destDir and returns the
// relative forward-slash paths of the files and symlinks it placed
// (directories are not recorded). It stages an already-extracted archive
// tree without re-reading the archive.
//
// Directories merge: they are created 0o755 on the destination side when
// missing and existing ones are left as they are. Files and symlinks replace
// whatever is at their destination path, so reinstalls overwrite cleanly;
// component archives may legitimately replace SDK static assets. Each leaf
// is renamed (with retry) and, across filesystems, copied instead. Symlinks
// must be relative and resolve inside srcDir; the check runs before anything
// at the destination is touched, so a rejected link never deletes an
// existing entry.
func MoveTree(srcDir, destDir string) ([]string, error) {
	info, err := os.Lstat(srcDir)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("move tree: %s is not a directory", srcDir)
	}
	op := &treeOp{root: srcDir, move: true}
	if err := op.placeDir(srcDir, destDir, ""); err != nil {
		return nil, err
	}
	return op.paths, nil
}

// CopyTree copies src, which may be a file, a symlink or a directory, to dst.
// It is the backup and restore primitive for component roots, so symlinks
// are reproduced as links with their original targets rather than validated
// or followed. Directory modes are preserved (applied after the directory is
// filled, so a read-only directory still receives its contents); files keep
// their modes. Existing files and symlinks at a destination path are
// replaced, existing directories are merged into.
func CopyTree(src, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	op := &treeOp{move: false}
	return op.place(src, dst, "")
}

// treeOp is one MoveTree or CopyTree run walking a source tree.
type treeOp struct {
	// root is the tree whose symlinks must stay inside it; empty means
	// links are copied verbatim.
	root string
	// move renames entries into place instead of copying them.
	move bool
	// paths collects the relative slash paths of placed files and symlinks.
	paths []string
}

func (op *treeOp) place(src, dst, rel string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.IsDir() {
		return op.placeDir(src, dst, rel)
	}
	if err := op.validateSymlink(src, info); err != nil {
		return err
	}
	if _, err := os.Lstat(dst); err == nil {
		if err := RemoveAllRetry(dst); err != nil {
			return fmt.Errorf("failed to overwrite %s: %w", dst, err)
		}
	}
	if err := op.placeLeaf(src, dst, info); err != nil {
		return err
	}
	if rel != "" {
		op.paths = append(op.paths, filepath.ToSlash(rel))
	}
	return nil
}

func (op *treeOp) placeDir(src, dst, rel string) error {
	if err := os.MkdirAll(dst, 0o755); err != nil {
		return err
	}
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if err := op.place(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name()), filepath.Join(rel, entry.Name())); err != nil {
			return err
		}
	}
	if op.move {
		return nil
	}
	// A backup reproduces the directory's mode; a merge into an existing
	// install tree leaves the destination's modes alone.
	info, err := os.Stat(src)
	if err != nil {
		return err
	}
	if perm := info.Mode().Perm(); perm != 0o755 {
		return os.Chmod(dst, perm)
	}
	return nil
}

// placeLeaf moves or copies one file or symlink whose destination is free.
func (op *treeOp) placeLeaf(src, dst string, info os.FileInfo) error {
	if op.move {
		if err := RenameRetry(src, dst); err == nil {
			return nil
		}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return placeSymlink(src, dst)
	}
	return CopyFile(src, dst, info.Mode())
}

// placeSymlink recreates the symlink src at dst with the same target. A link
// to a directory goes through SymlinkOrJunction so Windows without symlink
// privileges still gets a usable directory link. When a file link cannot be
// created (e.g. Windows without Developer Mode) the resolved file is copied
// instead of aborting the whole operation; a dangling link has nothing to
// copy, so the original symlink error is returned.
func placeSymlink(src, dst string) error {
	target, err := os.Readlink(src)
	if err != nil {
		return err
	}
	resolved, statErr := os.Stat(src)
	if statErr == nil && resolved.IsDir() {
		return SymlinkOrJunction(target, dst)
	}
	linkErr := os.Symlink(target, dst)
	if linkErr == nil || statErr != nil {
		return linkErr
	}
	return CopyFile(src, dst, resolved.Mode().Perm())
}

// validateSymlink rejects symlinks whose resolved target escapes op.root.
// Parent-traversing targets are allowed as long as the resolved path stays
// inside the root. It is a no-op when op.root is empty.
func (op *treeOp) validateSymlink(src string, info os.FileInfo) error {
	if op.root == "" || info.Mode()&os.ModeSymlink == 0 {
		return nil
	}
	target, err := os.Readlink(src)
	if err != nil {
		return err
	}
	if filepath.IsAbs(target) {
		return fmt.Errorf("refusing to create symlink with absolute target: %s -> %s", src, target)
	}
	resolved := filepath.Join(filepath.Dir(src), target)
	if !IsPathUnder(op.root, resolved) {
		return fmt.Errorf("refusing to create symlink whose target escapes the install root: %s -> %s", src, target)
	}
	return nil
}
