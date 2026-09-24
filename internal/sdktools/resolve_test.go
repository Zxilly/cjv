package sdktools

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveInstalledToolBinaryRequiresExistingBinary(t *testing.T) {
	tcDir := t.TempDir()

	_, err := ResolveInstalledToolBinary(tcDir, "cjc")
	var missing *cjverr.ToolNotInToolchainError
	require.ErrorAs(t, err, &missing)

	expectedPath := filepath.Join(tcDir, "bin", "cjc")
	if runtime.GOOS == "windows" {
		expectedPath += ".exe"
	}
	assert.Equal(t, "cjc", missing.Tool)
	assert.Equal(t, expectedPath, missing.Path)
}

func TestResolveInstalledToolBinaryReturnsPathWhenPresent(t *testing.T) {
	tcDir := t.TempDir()
	binDir := filepath.Join(tcDir, "tools", "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))

	toolPath := filepath.Join(binDir, "cjpm")
	if runtime.GOOS == "windows" {
		toolPath += ".exe"
	}
	require.NoError(t, os.WriteFile(toolPath, []byte("stub"), 0o755))

	resolved, err := ResolveInstalledToolBinary(tcDir, "cjpm")
	require.NoError(t, err)
	assert.Equal(t, toolPath, resolved)
}

func TestResolveInstalledToolBinaryPreservesUnknownToolError(t *testing.T) {
	_, err := ResolveInstalledToolBinary(t.TempDir(), "not-a-proxy")
	var unknown *cjverr.UnknownToolError
	require.True(t, errors.As(err, &unknown))
}

func TestResolveInstalledToolBinaryForTupleUsesTupleHostSuffix(t *testing.T) {
	tcDir := t.TempDir()
	winCJC := filepath.Join(tcDir, "bin", "cjc.exe")
	require.NoError(t, os.MkdirAll(filepath.Dir(winCJC), 0o755))
	require.NoError(t, os.WriteFile(winCJC, []byte("stub"), 0o755))

	resolved, err := ResolveInstalledToolBinaryForTuple(tcDir, "cjc", "win32-x64")
	require.NoError(t, err)
	assert.Equal(t, winCJC, resolved)

	_, err = ResolveInstalledToolBinaryForTuple(tcDir, "cjc", "linux-x64")
	var missing *cjverr.ToolNotInToolchainError
	require.ErrorAs(t, err, &missing)
	assert.Equal(t, filepath.Join(tcDir, "bin", "cjc"), missing.Path)
}
