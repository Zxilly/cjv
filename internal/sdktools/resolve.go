package sdktools

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/cjverr"
)

// ResolveToolBinary returns the full path to a tool binary within a toolchain directory.
func ResolveToolBinary(toolchainDir, toolName string) (string, error) {
	relPath := ToolRelativePath(toolName)
	if relPath == "" {
		return "", &cjverr.UnknownToolError{Name: toolName}
	}
	return PlatformBinaryName(filepath.Join(toolchainDir, relPath)), nil
}

// ResolveToolBinaryForTuple is ResolveToolBinary for a toolchain built for
// tuple, whose host OS decides the executable suffix instead of the running OS.
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
	return statInstalledBinary(toolName, binary)
}

// ResolveInstalledToolBinaryForTuple is ResolveInstalledToolBinary for a
// toolchain built for tuple.
func ResolveInstalledToolBinaryForTuple(toolchainDir, toolName, tuple string) (string, error) {
	binary, err := ResolveToolBinaryForTuple(toolchainDir, toolName, tuple)
	if err != nil {
		return "", err
	}
	return statInstalledBinary(toolName, binary)
}

func statInstalledBinary(toolName, binary string) (string, error) {
	if _, err := os.Stat(binary); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", &cjverr.ToolNotInToolchainError{Tool: toolName, Path: binary}
		}
		return "", err
	}
	return binary, nil
}
