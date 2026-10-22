package config

import (
	"context"
	"maps"
	"os"
	"path/filepath"
	"sync"

	"github.com/Zxilly/cjv/internal/fsops"
)

// SettingsFile caches reads for one invocation. Writes take a lock beside the
// settings file and merge against a fresh disk snapshot, so separate CJV_HOME
// values and separate processes cannot overwrite unrelated settings changes.
type SettingsFile struct {
	path   string
	cached *Settings
	mu     sync.Mutex
}

func NewSettingsFile(path string) *SettingsFile {
	return &SettingsFile{path: path}
}

func copySettings(s *Settings) *Settings {
	cp := *s
	cp.Overrides = maps.Clone(s.Overrides)
	return &cp
}

// Load returns a copy of the cached snapshot. Writes always re-read the file.
func (sf *SettingsFile) Load() (*Settings, error) {
	sf.mu.Lock()
	defer sf.mu.Unlock()
	if sf.cached == nil {
		s, err := loadSettingsSnapshot(sf.path)
		if err != nil {
			return nil, err
		}
		sf.cached = s
	}
	return copySettings(sf.cached), nil
}

// Save merges changes made to a loaded snapshot into the latest file. Fresh
// Settings values replace the complete file. Use Update to make an inherited
// value explicit, and the undo returned by UpdateWithUndo for rollback.
func (sf *SettingsFile) Save(s *Settings) error {
	_, _, err := sf.mutateValues(func(_ *Settings, values map[string]any) error {
		if s.snapshot == nil {
			clear(values)
			maps.Copy(values, settingsToSave(s))
			return nil
		}
		mergeSettingsChanges(values, settingsValues(s.snapshot.effective), settingsValues(s), false)
		return nil
	})
	return err
}

// Update persists explicit choices without copying inherited settings.
func (sf *SettingsFile) Update(update SettingsUpdate) (bool, error) {
	changed, _, err := sf.UpdateWithUndo(update)
	return changed, err
}

// UpdateWithUndo returns an undo that restores only this update's changes and
// only while each field/path still has the value this update wrote. A later
// writer's changes are retained. The undo is safe to call more than once.
func (sf *SettingsFile) UpdateWithUndo(update SettingsUpdate) (bool, func() error, error) {
	return sf.MutateWithUndo(func(*Settings) (SettingsUpdate, error) { return update, nil })
}

// Mutate decides an update from the latest effective settings under the file
// lock. The callback must not call back into this SettingsFile.
func (sf *SettingsFile) Mutate(fn func(*Settings) (SettingsUpdate, error)) (bool, error) {
	changed, _, err := sf.MutateWithUndo(fn)
	return changed, err
}

func (sf *SettingsFile) MutateWithUndo(fn func(*Settings) (SettingsUpdate, error)) (bool, func() error, error) {
	return sf.mutateValues(func(latest *Settings, values map[string]any) error {
		update, err := fn(copySettings(latest))
		if err != nil {
			return err
		}
		update.apply(values)
		return nil
	})
}

func (sf *SettingsFile) mutateValues(fn func(*Settings, map[string]any) error) (bool, func() error, error) {
	sf.mu.Lock()
	defer sf.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(sf.path), 0o755); err != nil {
		return false, nil, err
	}
	lock, err := fsops.LockFile(context.Background(), sf.path+".lock")
	if err != nil {
		return false, nil, err
	}
	defer lock.Close() //nolint:errcheck // releasing the lock cannot undo a published file
	latest, err := loadSettingsSnapshot(sf.path)
	if err != nil {
		return false, nil, err
	}
	before := latest.snapshot.persisted
	values := maps.Clone(before)
	if err := fn(latest, values); err != nil {
		return false, nil, err
	}
	if maps.EqualFunc(values, before, settingsValueEqual) {
		sf.cached = latest
		return false, func() error { return nil }, nil
	}
	if err := sf.saveLocked(values); err != nil {
		return false, nil, err
	}
	// Capture the exact persisted result, including presence, independently of
	// future mutations of this SettingsFile's cache.
	after := sf.cached.snapshot.persisted
	var undoMu sync.Mutex
	undone := false
	undo := func() error {
		undoMu.Lock()
		defer undoMu.Unlock()
		if undone {
			return nil
		}
		_, _, err := sf.mutateValues(func(_ *Settings, current map[string]any) error {
			mergeSettingsChanges(current, after, before, true)
			return nil
		})
		undone = err == nil
		return err
	}
	return true, undo, nil
}

func (sf *SettingsFile) saveLocked(values map[string]any) error {
	data, s, err := prepareSettingsValues(values)
	if err != nil {
		return err
	}
	if err := writeSettingsData(data, sf.path); err != nil {
		return err
	}
	sf.cached = s
	return nil
}

func (sf *SettingsFile) Invalidate() {
	sf.mu.Lock()
	defer sf.mu.Unlock()
	sf.cached = nil
}

func (sf *SettingsFile) Path() string { return sf.path }
