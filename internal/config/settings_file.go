package config

import (
	"maps"
	"sync"
)

// SettingsFile provides cached access to the settings file on disk.
// All reads return copies so callers can mutate freely, and Save
// writes to disk and updates the cache atomically.
type SettingsFile struct {
	path   string
	cached *Settings
	mu     sync.Mutex
}

// NewSettingsFile creates a SettingsFile backed by the given path.
func NewSettingsFile(path string) *SettingsFile {
	return &SettingsFile{path: path}
}

// copySettings returns a deep copy of s so callers can mutate freely.
func copySettings(s *Settings) *Settings {
	cp := *s
	if s.Overrides != nil {
		cp.Overrides = make(map[string]string, len(s.Overrides))
		maps.Copy(cp.Overrides, s.Overrides)
	}
	return &cp
}

// Load returns the settings, reading from disk on first call and
// returning a cached copy on subsequent calls. The returned pointer
// is a fresh copy that callers may modify.
func (sf *SettingsFile) Load() (*Settings, error) {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if sf.cached != nil {
		return copySettings(sf.cached), nil
	}

	s, _, err := LoadSettingsWithFallback(sf.path)
	if err != nil {
		return nil, err
	}
	sf.cached = s
	return copySettings(s), nil
}

// Save writes settings to disk and updates the in-memory cache.
func (sf *SettingsFile) Save(s *Settings) error {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if err := SaveSettings(s, sf.path); err != nil {
		return err
	}
	sf.cached = copySettings(s)
	return nil
}

// Invalidate clears the cache, forcing the next Load to re-read from disk.
func (sf *SettingsFile) Invalidate() {
	sf.mu.Lock()
	defer sf.mu.Unlock()
	sf.cached = nil
}

func (sf *SettingsFile) Path() string {
	return sf.path
}
