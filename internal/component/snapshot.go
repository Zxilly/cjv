package component

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"

	"github.com/Zxilly/cjv/internal/fsops"
)

type snapshot struct {
	tempDir string
	entries []snapshotEntry
}

type snapshotEntry struct {
	live    string
	backup  string
	existed bool
}

func takeSnapshot(roots Roots, names []Name) (*snapshot, error) {
	tempDir, err := os.MkdirTemp("", "cjv-component-snapshot-*")
	if err != nil {
		return nil, err
	}
	s := &snapshot{tempDir: tempDir}
	ok := false
	defer func() {
		if !ok {
			_ = s.cleanup()
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

func (s *snapshot) addPath(live, label string) error {
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
	if err := fsops.CopyTree(live, entry.backup); err != nil {
		return err
	}
	s.entries = append(s.entries, entry)
	return nil
}

func (s *snapshot) restore() error {
	var errs []error
	for i := len(s.entries) - 1; i >= 0; i-- {
		entry := s.entries[i]
		if err := fsops.RemoveAllRetry(entry.live); err != nil {
			errs = append(errs, err)
			continue
		}
		if !entry.existed {
			continue
		}
		if err := fsops.CopyTree(entry.backup, entry.live); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

func (s *snapshot) cleanup() error {
	if s == nil || s.tempDir == "" {
		return nil
	}
	return fsops.RemoveAllRetry(s.tempDir)
}
