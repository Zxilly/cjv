package component

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"

	"github.com/Zxilly/cjv/internal/utils"
)

type Snapshot struct {
	tempDir string
	entries []snapshotEntry
}

type snapshotEntry struct {
	live    string
	backup  string
	existed bool
}

func TakeSnapshot(roots Roots, names []Name) (*Snapshot, error) {
	tempDir, err := os.MkdirTemp("", "cjv-component-snapshot-*")
	if err != nil {
		return nil, err
	}
	s := &Snapshot{tempDir: tempDir}
	ok := false
	defer func() {
		if !ok {
			_ = s.Cleanup()
		}
	}()

	if err := s.addPath(metaPath(roots.TcDir), "meta"); err != nil {
		return nil, err
	}

	var rootsSeen []string
	for _, name := range names {
		spec, err := SpecFor(name)
		if err != nil {
			return nil, err
		}
		root := filepath.Clean(spec.InstallRoot(roots))
		if slices.Contains(rootsSeen, root) {
			continue
		}
		rootsSeen = append(rootsSeen, root)
		if err := s.addPath(root, fmt.Sprintf("root-%d", len(rootsSeen))); err != nil {
			return nil, err
		}
	}

	ok = true
	return s, nil
}

func (s *Snapshot) addPath(live, label string) error {
	entry := snapshotEntry{
		live:   live,
		backup: filepath.Join(s.tempDir, label),
	}
	if _, err := os.Lstat(live); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.entries = append(s.entries, entry)
			return nil
		}
		return err
	}
	entry.existed = true
	if err := copyTree(live, entry.backup); err != nil {
		return err
	}
	s.entries = append(s.entries, entry)
	return nil
}

func (s *Snapshot) Restore() error {
	var errs []error
	for i := len(s.entries) - 1; i >= 0; i-- {
		entry := s.entries[i]
		if err := utils.RemoveAllRetry(entry.live); err != nil {
			errs = append(errs, err)
			continue
		}
		if !entry.existed {
			continue
		}
		if err := copyTree(entry.backup, entry.live); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (s *Snapshot) Cleanup() error {
	if s == nil || s.tempDir == "" {
		return nil
	}
	return utils.RemoveAllRetry(s.tempDir)
}

func copyTree(src, dst string) error {
	info, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		target, err := os.Readlink(src)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return os.Symlink(target, dst)
	}
	if !info.IsDir() {
		if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
			return err
		}
		return utils.CopyFile(src, dst, info.Mode())
	}

	return filepath.WalkDir(src, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if entry.Type()&os.ModeSymlink != 0 {
			linkTarget, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			return os.Symlink(linkTarget, target)
		}
		if entry.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		return utils.CopyFile(path, target, info.Mode())
	})
}
