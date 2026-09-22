package config

import (
	"maps"

	"github.com/BurntSushi/toml"
)

// SettingsUpdate records explicit user choices. Nil fields are unchanged;
// non-nil fields are persisted even when equal to an inherited value. Empty
// strings and false are explicit values, not requests to inherit a fallback.
// A non-nil Overrides replaces the user's complete directory override map.
type SettingsUpdate struct {
	Home             *string
	DefaultToolchain *string
	ManifestURL      *string
	DistServer       *string
	AutoSelfUpdate   *string
	AutoInstall      *bool
	DefaultHost      *string
	Overrides        map[string]string
}

type settingsSnapshot struct {
	persisted map[string]any
	effective *Settings
}

func (u SettingsUpdate) apply(values map[string]any) {
	setSettingValue(values, "home", u.Home)
	setSettingValue(values, "default_toolchain", u.DefaultToolchain)
	setSettingValue(values, "manifest_url", u.ManifestURL)
	setSettingValue(values, "dist_server", u.DistServer)
	setSettingValue(values, "auto_self_update", u.AutoSelfUpdate)
	setSettingValue(values, "auto_install", u.AutoInstall)
	setSettingValue(values, "default_host", u.DefaultHost)
	if u.Overrides != nil {
		if len(u.Overrides) == 0 {
			delete(values, "overrides")
		} else {
			values["overrides"] = maps.Clone(u.Overrides)
		}
	}
}

func setSettingValue[T string | bool](values map[string]any, key string, value *T) {
	if value != nil {
		values[key] = *value
	}
}

func settingsValues(s *Settings) map[string]any {
	return map[string]any{
		"version":           currentSettingsVersion,
		"home":              s.Home,
		"default_toolchain": s.DefaultToolchain,
		"manifest_url":      s.ManifestURL,
		"dist_server":       s.DistServer,
		"auto_self_update":  s.AutoSelfUpdate,
		"auto_install":      s.AutoInstall,
		"default_host":      s.DefaultHost,
		"overrides":         maps.Clone(s.Overrides),
	}
}

func settingsValueEqual(a, b any) bool {
	if overrides, ok := a.(map[string]string); ok {
		other, ok := b.(map[string]string)
		return ok && maps.Equal(overrides, other)
	}
	return a == b
}

func loadSettingsSnapshot(path string) (*Settings, error) {
	s, meta, err := LoadSettingsWithFallback(path)
	if err != nil {
		return nil, err
	}
	return snapshotSettings(s, meta), nil
}

func snapshotSettings(s *Settings, meta toml.MetaData) *Settings {
	persisted := settingsValues(s)
	for key := range persisted {
		if key != "version" && !meta.IsDefined(key) {
			delete(persisted, key)
		}
	}
	if len(s.Overrides) == 0 {
		delete(persisted, "overrides")
	}
	s.snapshot = &settingsSnapshot{persisted: persisted, effective: copySettings(s)}
	return s
}

// settingsToSave preserves the provenance of a loaded snapshot. Fresh Settings
// values still describe a complete settings file, as they did before cached
// reads tracked provenance. Use Update to express an unchanged inherited value
// as an explicit user choice; a plain Settings assignment cannot express that.
func settingsToSave(s *Settings) map[string]any {
	current := settingsValues(s)
	if s.snapshot == nil {
		for _, key := range []string{"home", "dist_server", "default_host"} {
			if current[key] == "" {
				delete(current, key)
			}
		}
		if len(s.Overrides) == 0 {
			delete(current, "overrides")
		}
		return current
	}

	values := maps.Clone(s.snapshot.persisted)
	before := settingsValues(s.snapshot.effective)
	for key, value := range current {
		if !settingsValueEqual(value, before[key]) {
			values[key] = value
		}
	}
	if len(s.Overrides) == 0 {
		delete(values, "overrides")
	}
	return values
}
