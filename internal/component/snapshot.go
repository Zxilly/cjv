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
	roots   Roots
	names   []Name
}

type snapshotEntry struct {
	live           string
	backup         string
	existed        bool
	parent         string
	resolvedParent string
}

func takeSnapshot(roots Roots, names []Name) (*snapshot, error) {
	if err := checkEditRoots(roots, names); err != nil {
		return nil, err
	}
	tempDir, err := os.MkdirTemp("", "cjv-component-snapshot-*")
	if err != nil {
		return nil, err
	}
	s := &snapshot{tempDir: tempDir, roots: roots, names: append([]Name(nil), names...)}
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
	// Roots may intentionally live below a redirected CJV_HOME. Remember the
	// existing parent destination rather than rejecting those configurations;
	// rollback must not follow a newly redirected ancestor after apply fails.
	entry.parent = filepath.Dir(live)
	var err error
	entry.resolvedParent, err = resolveSnapshotParent(entry.parent)
	if err != nil {
		return err
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
	// An apply failure may follow a root or metadata directory being replaced
	// by a user link. Retain the backup rather than restoring through that link.
	if err := checkEditRoots(s.roots, s.names); err != nil {
		return err
	}
	for _, entry := range s.entries {
		resolved, err := resolveSnapshotParent(entry.parent)
		if err != nil {
			return err
		}
		if resolved != entry.resolvedParent {
			return fmt.Errorf("component restore parent changed: %s", entry.parent)
		}
	}
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

// Nonexistent parent directories are legitimate for a first installation.
// Resolve their closest existing ancestor while retaining the missing suffix,
// so a link created in that suffix during apply is also detected on rollback.
func resolveSnapshotParent(path string) (string, error) {
	parent := path
	for {
		resolved, err := filepath.EvalSymlinks(parent)
		if err == nil {
			rel, err := filepath.Rel(parent, path)
			if err != nil {
				return "", err
			}
			return filepath.Join(resolved, rel), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		next := filepath.Dir(parent)
		if next == parent {
			return "", err
		}
		parent = next
	}
}

func (s *snapshot) cleanup() error {
	if s == nil || s.tempDir == "" {
		return nil
	}
	return fsops.RemoveAllRetry(s.tempDir)
}
