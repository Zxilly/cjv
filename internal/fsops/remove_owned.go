package fsops

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// RemoveOwnedTree removes a disposable tree beneath root, including read-only
// directories copied from an SDK. It grants owner access only to directories
// being removed, never to files whose inode may be shared with another SDK.
// Neither intermediate directory links nor links inside the tree are followed.
// The root itself cannot be removed. Missing trees are already cleaned up.
func RemoveOwnedTree(root *os.Root, name string) error {
	if !filepath.IsLocal(name) || filepath.Clean(name) == "." {
		return fmt.Errorf("remove owned tree: invalid relative path %q", name)
	}
	return Retry(func() error {
		parent := root
		parts := strings.Split(filepath.Clean(name), string(filepath.Separator))
		for _, part := range parts[:len(parts)-1] {
			next, err := openOwnedDirectory(parent, part)
			if parent != root {
				_ = parent.Close()
			}
			if errors.Is(err, os.ErrNotExist) {
				return nil
			}
			if err != nil {
				return err
			}
			parent = next
		}
		if parent != root {
			defer parent.Close() //nolint:errcheck
		}
		return removeOwnedEntry(parent, parts[len(parts)-1])
	})
}

func openOwnedDirectory(parent *os.Root, name string) (*os.Root, error) {
	info, err := parent.Lstat(name)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("remove owned tree: %s must be a real directory", name)
	}
	dir, err := parent.OpenRoot(name)
	if err != nil {
		return nil, err
	}
	opened, err := dir.Stat(".")
	if err == nil && !os.SameFile(info, opened) {
		err = fmt.Errorf("remove owned tree: directory %s changed while opening", name)
	}
	if err != nil {
		_ = dir.Close()
		return nil, err
	}
	return dir, nil
}

func removeOwnedEntry(parent *os.Root, name string) error {
	info, err := parent.Lstat(name)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return parent.Remove(name)
	}
	dir, err := openOwnedDirectory(parent, name)
	if err != nil {
		return err
	}
	err = emptyOwnedDirectory(dir)
	if err := errors.Join(err, dir.Close()); err != nil {
		return err
	}
	return parent.Remove(name)
}

func emptyOwnedDirectory(dir *os.Root) error {
	info, err := dir.Stat(".")
	if err != nil {
		return err
	}
	if info.Mode().Perm()&0o700 != 0o700 {
		// The opened root pins the checked directory. Chmod of "." cannot
		// follow a replacement link at its former name.
		if err := dir.Chmod(".", info.Mode()|0o700); err != nil {
			return err
		}
	}
	f, err := dir.Open(".")
	if err != nil {
		return err
	}
	entries, readErr := f.ReadDir(-1)
	if err := errors.Join(readErr, f.Close()); err != nil {
		return err
	}
	for _, entry := range entries {
		if err := removeOwnedEntry(dir, entry.Name()); err != nil {
			return err
		}
	}
	return nil
}
