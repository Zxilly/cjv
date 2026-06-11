package proxy

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/env"
	"github.com/Zxilly/cjv/internal/toolchain"
)

const maxRecursion = 20

// ExtractToolName extracts the tool name from argv[0], stripping directory and .exe suffix.
func ExtractToolName(argv0 string) string {
	name := filepath.Base(argv0)
	if runtime.GOOS == "windows" {
		ext := filepath.Ext(name)
		if strings.EqualFold(ext, ".exe") {
			name = strings.TrimSuffix(name, ext)
		}
	}
	return name
}

func extractPlusToolchain(args []string) (string, []string, error) {
	tc, rest, present := toolchain.SplitPlusSelector(args)
	if present && tc == "" {
		return "", args, fmt.Errorf("toolchain name cannot be empty after '+'")
	}
	return tc, rest, nil
}

func checkRecursion(count int) error {
	if count >= maxRecursion {
		return &cjverr.RecursionLimitError{Max: maxRecursion}
	}
	return nil
}

// GetRecursionCount reads the recursion counter from the environment.
//
// cjv always writes a valid non-negative integer (see BuildProxyEnv). A value
// that fails to parse therefore means the counter was corrupted by something
// outside cjv; rather than silently resetting to 0 (which would defeat the
// infinite-proxy-loop guard), we fail safe by reporting the recursion limit so
// checkRecursion trips immediately.
func GetRecursionCount() int {
	s := os.Getenv(config.EnvRecursionCount)
	if s == "" {
		return 0
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		slog.Warn("invalid recursion counter; treating as recursion limit", "value", s)
		return maxRecursion
	}
	if n < 0 {
		return 0
	}
	return n
}

// ResolveToolBinary returns the full path to a tool binary within a toolchain directory.
func ResolveToolBinary(toolchainDir, toolName string) (string, error) {
	relPath := ToolRelativePath(toolName)
	if relPath == "" {
		return "", &cjverr.UnknownToolError{Name: toolName}
	}
	return PlatformBinaryName(filepath.Join(toolchainDir, relPath)), nil
}

func ResolveToolBinaryForTuple(toolchainDir, toolName, tuple string) (string, error) {
	relPath := ToolRelativePath(toolName)
	if relPath == "" {
		return "", &cjverr.UnknownToolError{Name: toolName}
	}
	return PlatformBinaryNameForTuple(filepath.Join(toolchainDir, relPath), tuple)
}

// ResolveInstalledToolBinary returns the full path to a proxy tool and verifies
// that the binary exists in the resolved toolchain.
func ResolveInstalledToolBinary(toolchainDir, toolName string) (string, error) {
	binary, err := ResolveToolBinary(toolchainDir, toolName)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(binary); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", &cjverr.ToolNotInToolchainError{Tool: toolName, Path: binary}
		}
		return "", err
	}
	return binary, nil
}

func ResolveInstalledToolBinaryForTuple(toolchainDir, toolName, tuple string) (string, error) {
	binary, err := ResolveToolBinaryForTuple(toolchainDir, toolName, tuple)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(binary); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", &cjverr.ToolNotInToolchainError{Tool: toolName, Path: binary}
		}
		return "", err
	}
	return binary, nil
}

// Run is the top-level entry point for proxy mode, called when argv[0] is a known tool name.
func Run(ctx context.Context, toolName string, args []string) error {
	count := GetRecursionCount()
	if err := checkRecursion(count); err != nil {
		return err
	}

	tcOverride, remainingArgs, err := extractPlusToolchain(args)
	if err != nil {
		return err
	}

	rt, err := env.ResolveRuntime(ctx, tcOverride, component.ApplyEnv)
	if err != nil {
		return err
	}

	binary, err := ResolveInstalledToolBinary(rt.Active.Dir, toolName)
	if err != nil {
		return err
	}

	proxyEnv := rt.ProxyEnv(os.Environ(), count)

	return execTool(ctx, binary, remainingArgs, proxyEnv)
}
