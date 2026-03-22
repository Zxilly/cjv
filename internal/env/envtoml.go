package env

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/Zxilly/cjv/internal/utils"
)

// EnvConfig represents the contents of env.toml
type EnvConfig struct {
	Vars        map[string]string `toml:"vars"`
	PathPrepend PathPrepend       `toml:"path_prepend"`
}

type PathPrepend struct {
	Entries []string `toml:"entries"`
}

// NewEnvConfig returns an initialized empty EnvConfig.
func NewEnvConfig() *EnvConfig {
	return &EnvConfig{Vars: make(map[string]string)}
}

// loadEnvConfigRaw loads an env.toml file, returning the raw error on failure
// (including os.ErrNotExist) so callers can distinguish missing from malformed.
func loadEnvConfigRaw(path string) (*EnvConfig, error) {
	var e EnvConfig
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if err := toml.Unmarshal(data, &e); err != nil {
		return nil, err
	}
	if e.Vars == nil {
		e.Vars = make(map[string]string)
	}
	return &e, nil
}

// LoadEnvConfig loads an env.toml file from path. Returns empty config if file doesn't exist.
func LoadEnvConfig(path string) (*EnvConfig, error) {
	cfg, err := loadEnvConfigRaw(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return NewEnvConfig(), nil
		}
		return nil, err
	}
	return cfg, nil
}

// LoadToolchainEnv loads env.toml from the toolchain directory, warns on
// parse errors (falling back to an empty config), and ensures the SDK
// library path is set.
func LoadToolchainEnv(ctx context.Context, tcDir string) *EnvConfig {
	envPath := filepath.Join(tcDir, "env.toml")
	cfg, err := loadEnvConfigRaw(envPath)
	if err == nil {
		EnsureLibraryPath(cfg, tcDir)
		return cfg
	}
	result := NewEnvConfig()
	if errors.Is(err, os.ErrNotExist) {
		captured, captureErr := CaptureEnvSetup(ctx, tcDir)
		if captureErr != nil {
			slog.Warn("failed to capture envsetup on demand", "toolchain", tcDir, "error", captureErr)
		} else {
			result = captured
		}
	} else {
		slog.Warn("failed to parse env.toml", "error", err)
	}
	EnsureLibraryPath(result, tcDir)
	return result
}

// Save writes the EnvConfig to path in TOML format.
func (e *EnvConfig) Save(path string) error {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(e); err != nil {
		return err
	}
	return utils.WriteFileAtomic(path, buf.Bytes(), 0o644)
}
