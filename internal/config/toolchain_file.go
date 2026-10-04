package config

import (
	"bytes"
	"fmt"
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
	Path       string   `toml:"path,omitempty"`
	Components []string `toml:"components,omitempty"`
	Targets    []string `toml:"targets,omitempty"`
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
	if tc.Toolchain.Path != "" {
		if tc.Toolchain.Channel != "" {
			return nil, fmt.Errorf("toolchain.channel and toolchain.path are mutually exclusive")
		}
		if !filepath.IsAbs(tc.Toolchain.Path) {
			return nil, fmt.Errorf("toolchain.path must be absolute")
		}
	}
	// Warn about unrecognized keys.
	for _, key := range meta.Undecoded() {
		slog.Warn("unrecognized key in toolchain file", "path", path, "key", key)
	}

	return &tc, nil
}
