package fsops

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// HardlinkIdenticalTree replaces identical regular payload files in a private
// staging tree with hardlinks. Metadata is always independent. Symlinks are
// never followed; unsupported links (including cross-volume links) keep copies.
// cjv mutations must replace/unlink payloads rather than write them in place.
func HardlinkIdenticalTree(source, staged string) error {
	return hardlinkIdenticalTree(source, staged, os.Link)
}
func hardlinkIdenticalTree(source, staged string, linkFile func(string, string) error) error {
	root, err := os.OpenRoot(source)
	if err != nil {
		return err
	}
	defer root.Close() //nolint:errcheck
	lbuf, rbuf := make([]byte, 64*1024), make([]byte, 64*1024)
	return filepath.WalkDir(staged, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(staged, path)
		if err != nil {
			return err
		}
		if rel == ".cjv" && entry.IsDir() {
			return filepath.SkipDir
		}
		// OpenRoot rejects parent symlinks escaping the source tree.
		info, err := root.Lstat(rel)
		if errors.Is(err, os.ErrNotExist) {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if err != nil {
			return err
		}
		if entry.IsDir() {
			// Inspect parents before visiting their children. A modified source
			// directory link loses deduplication, but never contaminates staging.
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		candidate := filepath.Join(source, rel)
		if !info.Mode().IsRegular() {
			return nil
		}
		stageInfo, err := entry.Info()
		if err != nil {
			return err
		}
		if info.Size() != stageInfo.Size() || info.Mode().Perm() != stageInfo.Mode().Perm() {
			return nil
		}
		equal, err := equalFiles(candidate, path, lbuf, rbuf)
		if err != nil {
			return err
		}
		if !equal {
			return nil
		}
		tmp, err := os.CreateTemp(filepath.Dir(path), ".cjv-link-*")
		if err != nil {
			return err
		}
		link := tmp.Name()
		if err := errors.Join(tmp.Close(), os.Remove(link)); err != nil {
			return err
		}
		if err := linkFile(candidate, link); err != nil {
			return nil
		}
		if err := RenameRetry(link, path); err != nil {
			// Windows rename cannot overwrite; remove the staged copy first.
			if err := os.Remove(path); err != nil {
				_ = os.Remove(link)
				return err
			}
			if err := RenameRetry(link, path); err != nil {
				return err
			}
		}
		return nil
	})
}

func equalFiles(a, b string, lbuf, rbuf []byte) (bool, error) {
	left, err := os.Open(a)
	if err != nil {
		return false, err
	}
	defer left.Close() //nolint:errcheck
	right, err := os.Open(b)
	if err != nil {
		return false, err
	}
	defer right.Close() //nolint:errcheck
	for {
		n, le := io.ReadFull(left, lbuf)
		m, re := io.ReadFull(right, rbuf)
		if le != nil && le != io.EOF && le != io.ErrUnexpectedEOF {
			return false, le
		}
		if re != nil && re != io.EOF && re != io.ErrUnexpectedEOF {
			return false, re
		}
		if n != m || !bytes.Equal(lbuf[:n], rbuf[:m]) {
			return false, nil
		}
		if le != nil || re != nil {
			return errors.Is(le, re), nil
		}
	}
}
