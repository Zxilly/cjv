package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
)

// NormalizePath returns a canonical absolute path for consistent comparison.
// On Windows, this uppercases the volume letter to avoid case mismatches.
func NormalizePath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		slog.Warn("failed to resolve absolute path, using original", "path", p, "error", err)
		return p
	}
	// Resolve symlinks for consistent matching
	if resolved, err := filepath.EvalSymlinks(abs); err == nil {
		abs = resolved
	}
	// Windows: normalize drive letter case (C:\ vs c:\)
	if runtime.GOOS == "windows" && len(abs) >= 2 && abs[1] == ':' {
		abs = strings.ToUpper(abs[:1]) + abs[1:]
	}
	return filepath.Clean(abs)
}

type OverrideSource int

const (
	SourceUnknown       OverrideSource = iota // invalid/unset
	SourceEnv                                 // CJV_TOOLCHAIN env var
	SourceOverride                            // directory override
	SourceToolchainFile                       // cangjie-sdk.toml
	SourceDefault                             // default toolchain
)

func (s OverrideSource) String() string {
	switch s {
	case SourceEnv:
		return "environment (CJV_TOOLCHAIN)"
	case SourceOverride:
		return "directory override"
	case SourceToolchainFile:
		return "cangjie-sdk.toml"
	case SourceDefault:
		return "default toolchain"
	default:
		return "unknown"
	}
}

// ResolveToolchain resolves the active toolchain by priority chain.
// The search walks up the directory tree and at each level checks both
// directory overrides and cangjie-sdk.toml (override takes precedence
// at the same level). A closer toolchain file wins over a farther
// directory override.
func ResolveToolchain(settings *Settings, cwd string) (string, OverrideSource, error) {
	// 1. Environment variable
	if env := os.Getenv(EnvToolchain); env != "" {
		return env, SourceEnv, nil
	}

	// Override keys are already stored in normalized form by "override set",
	// so we only need to normalize cwd for consistent matching — no per-key
	// NormalizePath syscalls on every resolution.

	// 2. Walk up from cwd, checking overrides AND toolchain file at each level
	dir := NormalizePath(cwd)
	for {
		// 2a. Check directory override at this level
		if tc, ok := settings.Overrides[dir]; ok {
			return tc, SourceOverride, nil
		}

		// 2b. Check cangjie-sdk.toml at this level
		candidate := filepath.Join(dir, ToolchainFileName)
		tc, parseErr := ParseToolchainFile(candidate)
		if parseErr != nil && !errors.Is(parseErr, os.ErrNotExist) {
			return "", SourceUnknown, fmt.Errorf("failed to parse %s: %w", candidate, parseErr)
		}
		if parseErr == nil {
			if tc.Toolchain.Channel != "" {
				return tc.Toolchain.Channel, SourceToolchainFile, nil
			}
			// File exists but channel is empty — report an error so users
			// know their toolchain file is incomplete rather than silently
			// falling through to a different toolchain.
			return "", SourceUnknown, fmt.Errorf("%s: toolchain.channel is empty; please specify a channel (e.g. lts, sts, nightly)", candidate)
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	// 3. Default toolchain
	if settings.DefaultToolchain != "" {
		return settings.DefaultToolchain, SourceDefault, nil
	}

	return "", SourceUnknown, &cjverr.NoToolchainConfiguredError{}
}
