//go:build integration

package integration

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/Zxilly/cjv/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// runCJV runs the cjv binary with CJV_NO_PATH_SETUP=1 to prevent PATH side-effects.
func runCJV(t *testing.T, binary string, cjvHome string, args ...string) (string, string, error) {
	return runCJVEnv(t, binary, cjvHome, []string{"CJV_NO_PATH_SETUP=1"}, args...)
}

// setupIntegrationEnv creates an isolated CJV_HOME with the mock server's
// manifest_url in settings.toml and the cjv binary pre-placed in bin/.
func setupIntegrationEnv(t *testing.T, binary string, serverURL string) string {
	t.Helper()
	cjvHome := t.TempDir()
	writeIntegrationSettings(t, cjvHome, serverURL)
	binDir := filepath.Join(cjvHome, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))

	// Pre-place the cjv binary in CJV_HOME/bin/ so proxy link creation works
	cjvName := "cjv"
	if runtime.GOOS == "windows" {
		cjvName = "cjv.exe"
	}
	copyBinary(t, binary, filepath.Join(binDir, cjvName))

	return cjvHome
}

func setupIntegrationEnvWithoutManagedBinary(t *testing.T, serverURL string) string {
	t.Helper()
	cjvHome := t.TempDir()
	writeIntegrationSettings(t, cjvHome, serverURL)
	return cjvHome
}

func writeIntegrationSettings(t *testing.T, cjvHome string, serverURL string) {
	t.Helper()
	settingsContent := fmt.Sprintf("manifest_url = %q\nauto_install = true\n", serverURL+"/sdk-versions.json")
	require.NoError(t, os.WriteFile(filepath.Join(cjvHome, "settings.toml"), []byte(settingsContent), 0o644))
}

func copyBinary(t *testing.T, src, dst string) {
	t.Helper()
	require.NoError(t, utils.CopyFile(src, dst, 0o755))
}

func TestIntegrationToolchainList(t *testing.T) {
	binary := buildCJV(t)
	cjvHome := t.TempDir()

	// Initially no toolchains installed
	stdout, _, err := runCJV(t, binary, cjvHome, "toolchain", "list")
	require.NoError(t, err)
	assert.Contains(t, stdout, "No toolchains installed")
}

func TestIntegrationOverrideSetUnset(t *testing.T) {
	binary := buildCJV(t)
	cjvHome := t.TempDir()

	// Ensure settings directory exists
	os.MkdirAll(cjvHome, 0o755)

	// Set override
	stdout, _, err := runCJV(t, binary, cjvHome, "override", "set", "lts-1.0.5")
	require.NoError(t, err)
	assert.Contains(t, stdout, "Override set")

	// List overrides
	stdout, _, err = runCJV(t, binary, cjvHome, "override", "list")
	require.NoError(t, err)
	assert.Contains(t, stdout, "lts-1.0.5")
}

func TestIntegrationShowHome(t *testing.T) {
	binary := buildCJV(t)
	cjvHome := t.TempDir()

	stdout, _, err := runCJV(t, binary, cjvHome, "show", "home")
	require.NoError(t, err)
	assert.Contains(t, stdout, cjvHome)
}

func TestIntegrationMockServer(t *testing.T) {
	// Verify mock server starts and serves manifest
	server := testutil.MockDistServer(t)
	require.NotNil(t, server)
	require.NotEmpty(t, server.URL)
}

func TestIntegrationCompletionsBash(t *testing.T) {
	binary := buildCJV(t)
	cjvHome := t.TempDir()

	stdout, _, err := runCJV(t, binary, cjvHome, "completion", "bash")
	require.NoError(t, err)
	assert.Contains(t, stdout, "bash")
}

func TestIntegrationVersion(t *testing.T) {
	binary := buildCJV(t)
	cjvHome := t.TempDir()

	stdout, _, err := runCJV(t, binary, cjvHome, "--version")
	require.NoError(t, err)
	assert.Contains(t, stdout, "cjv")
}

// TestIntegrationInstallFlow tests the full lifecycle:
// install -> show active -> which -> default -> uninstall
func TestIntegrationInstallFlow(t *testing.T) {
	binary := buildCJV(t)
	server := testutil.MockDistServer(t)
	cjvHome := setupIntegrationEnv(t, binary, server.URL)

	// 1. Install lts (resolves to lts-1.0.5 from mock manifest)
	stdout, stderr, err := runCJV(t, binary, cjvHome, "install", "lts")
	require.NoError(t, err, "install failed: stdout=%s stderr=%s", stdout, stderr)
	assert.Contains(t, stdout, "lts-1.0.5")

	// 2. Verify toolchain directory was created
	tcDir := filepath.Join(cjvHome, "toolchains", "lts-1.0.5")
	_, err = os.Stat(tcDir)
	require.NoError(t, err, "toolchain directory should exist after install")

	// 3. Verify bin/cjc stub exists in the installed toolchain
	cjcPath := filepath.Join(tcDir, "bin", "cjc")
	if runtime.GOOS == "windows" {
		cjcPath += ".exe"
	}
	_, err = os.Stat(cjcPath)
	require.NoError(t, err, "cjc stub should exist in toolchain bin/")

	// 4. Show active — should report lts-1.0.5 as default
	stdout, _, err = runCJV(t, binary, cjvHome, "show", "active")
	require.NoError(t, err)
	assert.Contains(t, stdout, "lts-1.0.5")

	// 5. Toolchain list should include lts-1.0.5
	stdout, _, err = runCJV(t, binary, cjvHome, "toolchain", "list")
	require.NoError(t, err)
	assert.Contains(t, stdout, "lts-1.0.5")

	// 6. Which cjc — should return a path ending with bin/cjc
	stdout, stderr, err = runCJV(t, binary, cjvHome, "which", "cjc")
	require.NoError(t, err, "which failed: stdout=%s stderr=%s", stdout, stderr)
	whichOutput := strings.TrimSpace(stdout + stderr)
	assert.Contains(t, whichOutput, "bin")
	assert.Contains(t, whichOutput, "cjc")

	// 7. Uninstall
	stdout, stderr, err = runCJV(t, binary, cjvHome, "uninstall", "lts-1.0.5")
	require.NoError(t, err, "uninstall failed: stdout=%s stderr=%s", stdout, stderr)

	// 8. Verify toolchain directory is gone
	_, err = os.Stat(tcDir)
	assert.True(t, errors.Is(err, os.ErrNotExist), "toolchain directory should be removed after uninstall")

	// 9. Toolchain list should be empty again
	stdout, _, err = runCJV(t, binary, cjvHome, "toolchain", "list")
	require.NoError(t, err)
	assert.Contains(t, stdout, "No toolchains installed")
}

func TestIntegrationInstallBootstrapsManagedBinaryAndSelfUpdate(t *testing.T) {
	binary := buildCJV(t)
	server := testutil.MockDistServer(t)
	cjvHome := setupIntegrationEnvWithoutManagedBinary(t, server.URL)

	managedBinary := filepath.Join(cjvHome, "bin", "cjv")
	if runtime.GOOS == "windows" {
		managedBinary += ".exe"
	}
	_, err := os.Stat(managedBinary)
	assert.True(t, errors.Is(err, os.ErrNotExist))

	stdout, stderr, err := runCJV(t, binary, cjvHome, "install", "lts")
	require.NoError(t, err, "install failed: stdout=%s stderr=%s", stdout, stderr)
	assert.FileExists(t, managedBinary)

	stdout, stderr, err = runCJV(t, binary, cjvHome, "self", "update")
	require.NoError(t, err, "self update failed: stdout=%s stderr=%s", stdout, stderr)
}

// TestIntegrationDefaultCommand tests setting and switching the default toolchain.
func TestIntegrationDefaultCommand(t *testing.T) {
	binary := buildCJV(t)
	server := testutil.MockDistServer(t)
	cjvHome := setupIntegrationEnv(t, binary, server.URL)

	// Install lts (auto-sets as default)
	_, _, err := runCJV(t, binary, cjvHome, "install", "lts")
	require.NoError(t, err)

	// Show active should report lts-1.0.5
	stdout, _, err := runCJV(t, binary, cjvHome, "show", "active")
	require.NoError(t, err)
	assert.Contains(t, stdout, "lts-1.0.5")

	// Set default to a different name (even if not installed, the command just saves it)
	stdout, _, err = runCJV(t, binary, cjvHome, "default", "nightly-20250101")
	require.NoError(t, err)
	assert.Contains(t, stdout, "nightly-20250101")

	// Show active should now report nightly-20250101
	stdout, _, err = runCJV(t, binary, cjvHome, "show", "active")
	require.NoError(t, err)
	assert.Contains(t, stdout, "nightly-20250101")
}

// TestIntegrationInstallAlreadyInstalled tests that re-installing
// an already installed toolchain is a no-op.
func TestIntegrationInstallAlreadyInstalled(t *testing.T) {
	binary := buildCJV(t)
	server := testutil.MockDistServer(t)
	cjvHome := setupIntegrationEnv(t, binary, server.URL)

	// Install lts
	_, _, err := runCJV(t, binary, cjvHome, "install", "lts")
	require.NoError(t, err)

	// Install again — should report already installed
	stdout, _, err := runCJV(t, binary, cjvHome, "install", "lts")
	require.NoError(t, err)
	assert.Contains(t, stdout, "already installed")
}

// TestIntegrationShowInstalledMultiple tests show installed with actual toolchains.
func TestIntegrationShowInstalledMultiple(t *testing.T) {
	binary := buildCJV(t)
	server := testutil.MockDistServer(t)
	cjvHome := setupIntegrationEnv(t, binary, server.URL)

	// Install lts
	_, _, err := runCJV(t, binary, cjvHome, "install", "lts")
	require.NoError(t, err)

	// Show installed
	stdout, _, err := runCJV(t, binary, cjvHome, "show", "installed")
	require.NoError(t, err)
	assert.Contains(t, stdout, "lts-1.0.5")
}

// TestIntegrationWhichUnknownTool tests that 'which' for an unknown tool returns an error.
func TestIntegrationWhichUnknownTool(t *testing.T) {
	binary := buildCJV(t)
	server := testutil.MockDistServer(t)
	cjvHome := setupIntegrationEnv(t, binary, server.URL)

	// Install a toolchain first so active toolchain is set
	_, _, err := runCJV(t, binary, cjvHome, "install", "lts")
	require.NoError(t, err)

	// which for unknown tool
	_, _, err = runCJV(t, binary, cjvHome, "which", "nonexistent-tool")
	assert.Error(t, err, "'which' for unknown tool should return an error")
}

// TestIntegrationUninstallClearsDefault tests that uninstalling the default
// toolchain clears the default_toolchain setting.
func TestIntegrationUninstallClearsDefault(t *testing.T) {
	binary := buildCJV(t)
	server := testutil.MockDistServer(t)
	cjvHome := setupIntegrationEnv(t, binary, server.URL)

	// Install lts (auto-sets as default)
	_, _, err := runCJV(t, binary, cjvHome, "install", "lts")
	require.NoError(t, err)

	// Uninstall
	_, _, err = runCJV(t, binary, cjvHome, "uninstall", "lts-1.0.5")
	require.NoError(t, err)

	// Show active should fail (no toolchain configured)
	_, _, err = runCJV(t, binary, cjvHome, "show", "active")
	assert.Error(t, err, "show active should fail after uninstalling the default toolchain")
}
