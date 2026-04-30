package proxy

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/env"
	"github.com/Zxilly/cjv/internal/resolve"
)

const maxRecursion = 20

// ExtractToolName extracts the tool name from argv[0], stripping directory and .exe suffix.
func ExtractToolName(argv0 string) string {
	name := filepath.Base(argv0)
	if runtime.GOOS == "windows" {
		name = strings.TrimSuffix(name, ".exe")
	}
	return name
}

func extractPlusToolchain(args []string) (string, []string, error) {
	if len(args) > 0 && strings.HasPrefix(args[0], "+") {
		tc := args[0][1:]
		if tc == "" {
			return "", args, fmt.Errorf("toolchain name cannot be empty after '+'")
		}
		return tc, args[1:], nil
	}
	return "", args, nil
}

func checkRecursion(count int) error {
	if count >= maxRecursion {
		return &cjverr.RecursionLimitError{Max: maxRecursion}
	}
	return nil
}

// GetRecursionCount reads the recursion counter from the environment.
func GetRecursionCount() int {
	s := os.Getenv(config.EnvRecursionCount)
	if s == "" {
		return 0
	}
	n, _ := strconv.Atoi(s)
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

	active, err := resolve.Active(ctx, tcOverride)
	if err != nil {
		return err
	}

	binary, err := ResolveInstalledToolBinary(active.Dir, toolName)
	if err != nil {
		return err
	}

	envCfg := env.LoadToolchainEnv(ctx, active.Dir)

	binDir, err := config.BinDir()
	if err != nil {
		return fmt.Errorf("failed to determine bin directory: %w", err)
	}
	proxyEnv := env.BuildProxyEnv(os.Environ(), env.ProxyEnvContext{
		Cfg:             envCfg,
		CjvBinDir:       binDir,
		ToolchainBinDir: filepath.Join(active.Dir, "bin"),
		Recursion:       count,
		ToolchainName:   active.Name,
	})

	return execTool(ctx, binary, remainingArgs, proxyEnv)
}
