package config

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
)

const EnvFallbackSettings = "CJV_FALLBACK_SETTINGS"

// DefaultFallbackPath returns the platform-specific fallback settings path.
// The CJV_FALLBACK_SETTINGS environment variable overrides the default.
func DefaultFallbackPath() string {
	if p := os.Getenv(EnvFallbackSettings); p != "" {
		return p
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("ProgramData"), "cjv", "settings.toml")
	}
	return "/etc/cjv/settings.toml"
}

// LoadSettingsWithFallback loads user settings and merges undefined fields from the fallback file.
func LoadSettingsWithFallback(userPath string) (*Settings, toml.MetaData, error) {
	s, meta, err := loadSettingsWithMeta(userPath)
	if err != nil {
		return nil, toml.MetaData{}, err
	}

	fbPath := DefaultFallbackPath()
	fb, fbErr := loadFallbackSettings(fbPath)
	if fbErr != nil {
		if errors.Is(fbErr, os.ErrNotExist) {
			slog.Debug("fallback settings not found", "path", fbPath)
		} else {
			slog.Warn("failed to load fallback settings", "path", fbPath, "error", fbErr)
		}
		return s, meta, nil
	}

	mergeFromFallback(s, fb, meta)
	return s, meta, nil
}

// loadFallbackSettings loads settings from the fallback path.
// Unlike LoadSettings, it returns os.ErrNotExist directly instead of silently returning defaults.
func loadFallbackSettings(path string) (*Settings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	s := DefaultSettings()
	if err := toml.Unmarshal(data, &s); err != nil {
		return nil, err
	}
	if s.Overrides == nil {
		s.Overrides = make(map[string]string)
	}
	return &s, nil
}

// mergeFromFallback fills user-undefined fields from the fallback settings.
// Only fields not explicitly set in the user's TOML file are merged.
func mergeFromFallback(user, fallback *Settings, meta toml.MetaData) {
	if !meta.IsDefined("default_toolchain") && fallback.DefaultToolchain != "" {
		user.DefaultToolchain = fallback.DefaultToolchain
	}
	if !meta.IsDefined("manifest_url") && fallback.ManifestURL != "" {
		user.ManifestURL = fallback.ManifestURL
	}
	if !meta.IsDefined("auto_self_update") && fallback.AutoSelfUpdate != "" {
		user.AutoSelfUpdate = fallback.AutoSelfUpdate
	}
	if !meta.IsDefined("auto_install") {
		user.AutoInstall = fallback.AutoInstall
	}
	if !meta.IsDefined("default_host") && fallback.DefaultHost != "" {
		user.DefaultHost = fallback.DefaultHost
	}
	if !meta.IsDefined("profile") && fallback.Profile != "" {
		user.Profile = fallback.Profile
	}
}
