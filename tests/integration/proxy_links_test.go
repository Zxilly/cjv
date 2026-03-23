//go:build integration

package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationInitCreatesAllProxyLinks verifies that cjv init creates
// symlinks/copies for every proxy tool in CJV_HOME/bin/.
func TestIntegrationInitCreatesAllProxyLinks(t *testing.T) {
	binary := buildCJV(t)
	cjvHome := t.TempDir()

	stdout, stderr, err := runCJV(t, binary, cjvHome, "init", "-y", "--default-toolchain", "none", "--no-modify-path")
	require.NoError(t, err, "init failed: stdout=%s stderr=%s", stdout, stderr)

	binDir := filepath.Join(cjvHome, "bin")
	for _, tool := range proxy.AllProxyTools() {
		toolPath := filepath.Join(binDir, proxy.PlatformBinaryName(tool))
		assert.FileExists(t, toolPath, "proxy link for %q should exist after init", tool)

		info, err := os.Stat(toolPath)
		if err == nil {
			assert.Greater(t, info.Size(), int64(0),
				"proxy link for %q should be non-empty", tool)
		}
	}
}

// TestIntegrationInstallCreatesProxyLinks verifies that cjv install also
// creates/updates proxy links in CJV_HOME/bin/.
func TestIntegrationInstallCreatesProxyLinks(t *testing.T) {
	binary := buildCJV(t)
	server := testutil.MockDistServer(t)
	cjvHome := setupIntegrationEnv(t, binary, server.URL)

	stdout, stderr, err := runCJV(t, binary, cjvHome, "install", "lts")
	require.NoError(t, err, "install failed: stdout=%s stderr=%s", stdout, stderr)

	binDir := filepath.Join(cjvHome, "bin")
	for _, tool := range proxy.AllProxyTools() {
		toolPath := filepath.Join(binDir, proxy.PlatformBinaryName(tool))
		assert.FileExists(t, toolPath, "proxy link for %q should exist after install", tool)
	}
}

// TestIntegrationInitRerunRestoresProxyLinks verifies that running init again
// restores a corrupted proxy link.
func TestIntegrationInitRerunRestoresProxyLinks(t *testing.T) {
	binary := buildCJV(t)
	cjvHome := t.TempDir()

	// First init
	stdout, stderr, err := runCJV(t, binary, cjvHome, "init", "-y", "--default-toolchain", "none", "--no-modify-path")
	require.NoError(t, err, "first init failed: stdout=%s stderr=%s", stdout, stderr)

	// Corrupt one proxy link
	cjcPath := filepath.Join(cjvHome, "bin", proxy.PlatformBinaryName("cjc"))
	require.NoError(t, os.WriteFile(cjcPath, []byte{}, 0o755))
	info, _ := os.Stat(cjcPath)
	require.Equal(t, int64(0), info.Size(), "cjc should be corrupted (empty)")

	// Re-init
	stdout, stderr, err = runCJV(t, binary, cjvHome, "init", "-y", "--default-toolchain", "none", "--no-modify-path")
	require.NoError(t, err, "re-init failed: stdout=%s stderr=%s", stdout, stderr)

	// Verify restored
	info, err = os.Stat(cjcPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0), "cjc proxy link should be restored after re-init")
}
