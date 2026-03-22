package proxy

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for CreateAllProxyLinks — creates symlinks/copies for all
// proxy tools (cjc, cjpm, etc.) pointing to the cjv binary.

func TestCreateAllProxyLinks_CreatesAllTools(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	binDir := filepath.Join(home, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))

	// Create the cjv binary that all tools will link to
	cjvPath := filepath.Join(binDir, CjvBinaryName())
	require.NoError(t, os.WriteFile(cjvPath, []byte("cjv-binary"), 0o755))

	require.NoError(t, CreateAllProxyLinks())

	// Verify all proxy tools were created
	for _, tool := range AllProxyTools() {
		toolPath := filepath.Join(binDir, PlatformBinaryName(tool))
		assert.FileExists(t, toolPath,
			"proxy link for %q should exist", tool)
	}
}

func TestCjvBinaryName_HasCorrectExtension(t *testing.T) {
	name := CjvBinaryName()
	assert.Equal(t, PlatformBinaryName("cjv"), name)
}
