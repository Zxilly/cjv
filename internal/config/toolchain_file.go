package config

import (
	"bytes"
	"log/slog"
	"os"

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
