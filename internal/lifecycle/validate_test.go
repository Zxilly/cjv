package lifecycle

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for validateInstallation -- verifies that an extracted SDK
// contains the essential "cjc" binary.

func TestValidateInstallation_ValidSDK(t *testing.T) {
	dir := t.TempDir()

	// Create the cjc binary at the expected location
	relPath := sdktools.ToolRelativePath("cjc")
	cjcPath := filepath.Join(dir, relPath)
	if runtime.GOOS == "windows" {
		cjcPath += ".exe"
	}
	require.NoError(t, os.MkdirAll(filepath.Dir(cjcPath), 0o755))
	require.NoError(t, os.WriteFile(cjcPath, []byte("stub"), 0o755))

	assert.NoError(t, validateInstallation(dir, ""),
		"should pass when cjc binary exists at the expected path")
}

func TestValidateInstallationUsesResolvedTupleBinaryName(t *testing.T) {
	winDir := t.TempDir()
	winCJCPath := filepath.Join(winDir, sdktools.ToolRelativePath("cjc")) + ".exe"
	require.NoError(t, os.MkdirAll(filepath.Dir(winCJCPath), 0o755))
	require.NoError(t, os.WriteFile(winCJCPath, []byte("stub"), 0o755))
	assert.NoError(t, validateInstallation(winDir, "win32-x64"))

	linuxDir := t.TempDir()
	linuxCJCPath := filepath.Join(linuxDir, sdktools.ToolRelativePath("cjc"))
	require.NoError(t, os.MkdirAll(filepath.Dir(linuxCJCPath), 0o755))
	require.NoError(t, os.WriteFile(linuxCJCPath, []byte("stub"), 0o755))
	assert.NoError(t, validateInstallation(linuxDir, "linux-x64"))
}

func TestValidateInstallation_MissingBinary(t *testing.T) {
	dir := t.TempDir()
	// Empty directory — no cjc binary

	err := validateInstallation(dir, "")
	assert.Error(t, err, "should fail when cjc binary is missing")
}

func TestValidateInstallation_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	err := validateInstallation(dir, "")
	assert.Error(t, err, "should fail on empty directory")
}
