package config

import (
	"bytes"
	"errors"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const ToolchainFileName = "cangjie-sdk.toml"

type ToolchainFileContent struct {
	Toolchain ToolchainSection `toml:"toolchain"`
}

type ToolchainSection struct {
	Channel    string   `toml:"channel"`
	Components []string `toml:"components,omitempty"`
	Targets    []string `toml:"targets,omitempty"`
	Profile    string   `toml:"profile,omitempty"`
}

func ParseToolchainFile(path string) (*ToolchainFileContent, error) {
	var tc ToolchainFileContent
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	meta, err := toml.NewDecoder(bytes.NewReader(data)).Decode(&tc)
	if err != nil {
		return nil, err
	}

	// Warn about unrecognized keys (e.g. typos like [toolchian] or channal = "lts").
	for _, key := range meta.Undecoded() {
		slog.Warn("unrecognized key in toolchain file", "path", path, "key", key)
	}

	return &tc, nil
}

// FindToolchainFile walks up from startDir looking for cangjie-sdk.toml.
func FindToolchainFile(startDir string) (string, error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return "", err
	}
	for {
		candidate := filepath.Join(dir, ToolchainFileName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", nil // reached filesystem root
		}
		dir = parent
	}
}
