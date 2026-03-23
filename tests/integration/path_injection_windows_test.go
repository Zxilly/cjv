//go:build integration && windows

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationInitPathSetupWindowsRegistry verifies that cjv init adds
// CJV_HOME\bin to HKCU\Environment\Path.
func TestIntegrationInitPathSetupWindowsRegistry(t *testing.T) {
	requireCI(t)
	binary := buildCJV(t)
	cjvHome := t.TempDir()

	stdout, stderr, err := runCJVEnv(t, binary, cjvHome, nil,
		"init", "-y", "--default-toolchain", "none")
	require.NoError(t, err, "init failed: stdout=%s stderr=%s", stdout, stderr)

	expectedBinDir := filepath.Join(cjvHome, "bin")
	registryPath, err := testutil.ReadRegistryPath()
	require.NoError(t, err, "failed to read registry PATH")
	assert.Contains(t, strings.ToLower(registryPath), strings.ToLower(expectedBinDir),
		"registry PATH should contain CJV_HOME\\bin after init")

	// Env scripts should also exist
	assert.FileExists(t, filepath.Join(cjvHome, "env"))
	assert.FileExists(t, filepath.Join(cjvHome, "env.ps1"))
}

// TestIntegrationInitPathSetupWindowsIdempotent verifies that running init
// twice does not duplicate the bin directory in the registry PATH.
func TestIntegrationInitPathSetupWindowsIdempotent(t *testing.T) {
	requireCI(t)
	binary := buildCJV(t)
	cjvHome := t.TempDir()

	// Run init twice
	_, _, err := runCJVEnv(t, binary, cjvHome, nil,
		"init", "-y", "--default-toolchain", "none")
	require.NoError(t, err)

	_, _, err = runCJVEnv(t, binary, cjvHome, nil,
		"init", "-y", "--default-toolchain", "none")
	require.NoError(t, err)

	expectedBinDir := filepath.Join(cjvHome, "bin")
	registryPath, err := testutil.ReadRegistryPath()
	require.NoError(t, err)

	entries := strings.Split(registryPath, string(os.PathListSeparator))
	count := 0
	for _, entry := range entries {
		if strings.EqualFold(entry, expectedBinDir) {
			count++
		}
	}
	assert.Equal(t, 1, count,
		"CJV_HOME\\bin should appear exactly once in registry PATH")
}

// TestIntegrationInitNoModifyPathSkipsWindows verifies that --no-modify-path
// does not modify the Windows registry PATH.
func TestIntegrationInitNoModifyPathSkipsWindows(t *testing.T) {
	binary := buildCJV(t)
	cjvHome := t.TempDir()

	pathBefore, err := testutil.ReadRegistryPath()
	require.NoError(t, err)

	_, _, err = runCJVEnv(t, binary, cjvHome, nil,
		"init", "-y", "--default-toolchain", "none", "--no-modify-path")
	require.NoError(t, err)

	pathAfter, err := testutil.ReadRegistryPath()
	require.NoError(t, err)
	assert.Equal(t, pathBefore, pathAfter,
		"registry PATH should be unchanged with --no-modify-path")

	// Env scripts should still be written
	assert.FileExists(t, filepath.Join(cjvHome, "env"))
	assert.FileExists(t, filepath.Join(cjvHome, "env.ps1"))
}

// TestIntegrationInitEnvScriptsWindows verifies that cjv init writes env and
// env.ps1 scripts on Windows.
func TestIntegrationInitEnvScriptsWindows(t *testing.T) {
	binary := buildCJV(t)
	cjvHome := t.TempDir()

	stdout, stderr, err := runCJVEnv(t, binary, cjvHome, nil,
		"init", "-y", "--default-toolchain", "none", "--no-modify-path")
	require.NoError(t, err, "init failed: stdout=%s stderr=%s", stdout, stderr)

	expectedBinDir := filepath.Join(cjvHome, "bin")

	envContent, err := os.ReadFile(filepath.Join(cjvHome, "env"))
	require.NoError(t, err, "env script should exist")
	assert.Contains(t, string(envContent), expectedBinDir)

	ps1Content, err := os.ReadFile(filepath.Join(cjvHome, "env.ps1"))
	require.NoError(t, err, "env.ps1 script should exist")
	assert.Contains(t, string(ps1Content), expectedBinDir)
	assert.Contains(t, string(ps1Content), "$env:PATH")
}
