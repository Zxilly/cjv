package config

import (
	"bytes"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"github.com/BurntSushi/toml"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/utils"
)

const DefaultManifestURL = "https://raw.githubusercontent.com/Zxilly/setup-cangjie/master/sdk-versions.json"

// AutoSelfUpdate controls self-update behavior.
const (
	AutoSelfUpdateEnable  = "enable"
	AutoSelfUpdateDisable = "disable"
	AutoSelfUpdateCheck   = "check"
)

// ValidAutoSelfUpdate returns true if s is a valid AutoSelfUpdate value.
func ValidAutoSelfUpdate(s string) bool {
	return s == AutoSelfUpdateEnable || s == AutoSelfUpdateDisable || s == AutoSelfUpdateCheck
}

// currentSettingsVersion is the latest settings format version this binary understands.
const currentSettingsVersion = 1

type Settings struct {
	Version          int               `toml:"version"`
	DefaultToolchain string            `toml:"default_toolchain"`
	ManifestURL      string            `toml:"manifest_url"`
	AutoSelfUpdate   string            `toml:"auto_self_update"`
	AutoInstall      bool              `toml:"auto_install"`
	DefaultHost      string            `toml:"default_host,omitempty"`
	Profile          string            `toml:"profile,omitempty"`
	Overrides        map[string]string `toml:"overrides,omitempty"`
}

func DefaultSettings() Settings {
	return Settings{
		Version:        currentSettingsVersion,
		ManifestURL:    DefaultManifestURL,
		AutoSelfUpdate: AutoSelfUpdateCheck,
		AutoInstall:    true,
		Overrides:      make(map[string]string),
	}
}

// LoadSettings loads settings from file, returning defaults if the file doesn't exist.
func LoadSettings(path string) (*Settings, error) {
	s, _, err := loadSettingsWithMeta(path)
	return s, err
}

// loadSettingsWithMeta loads settings and returns the TOML metadata for field-level inspection.
func loadSettingsWithMeta(path string) (*Settings, toml.MetaData, error) {
	s := DefaultSettings()
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &s, toml.MetaData{}, nil
		}
		return nil, toml.MetaData{}, fmt.Errorf("failed to read settings from %s: %w", path, err)
	}
	md, err := toml.NewDecoder(bytes.NewReader(data)).Decode(&s)
	if err != nil {
		return nil, toml.MetaData{}, fmt.Errorf("failed to parse settings from %s: %w", path, err)
	}
	for _, key := range md.Undecoded() {
		slog.Warn(i18n.T("UnknownSettingsField", i18n.MsgData{
			"Field": key.String(),
		}))
	}
	if s.Overrides == nil {
		s.Overrides = make(map[string]string)
	}
	// Restore default if ManifestURL was set to empty string in config file
	if s.ManifestURL == "" {
		s.ManifestURL = DefaultManifestURL
	}
	// in memory only; missing version field defaults to v1 without a disk write
	if s.Version == 0 {
		s.Version = 1
	}
	if s.Version > currentSettingsVersion {
		return nil, toml.MetaData{}, fmt.Errorf("%s", i18n.T("UnsupportedSettingsVersion", i18n.MsgData{
			"Version": strconv.Itoa(s.Version),
		}))
	}
	migrateSettings(&s)
	return &s, md, nil
}

// migrateSettings applies in-memory migrations from older settings versions
// to the current version. Add new cases as currentSettingsVersion is bumped.
func migrateSettings(s *Settings) {
	// Example for future migration:
	//   if s.Version < 2 { /* migrate v1 → v2 */ s.Version = 2 }
	s.Version = currentSettingsVersion
}

// Prefer SettingsFile.Save when a cached SettingsFile is available.
func SaveSettings(s *Settings, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	enc := toml.NewEncoder(&buf)
	if err := enc.Encode(s); err != nil {
		return err
	}
	return utils.WriteFileAtomic(path, buf.Bytes(), 0o644)
}
