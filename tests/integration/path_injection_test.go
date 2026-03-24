//go:build integration && !windows

package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegrationInitPathSetupUnix verifies that cjv init writes PATH blocks
// into shell config files (.bashrc, .zshrc, .profile, .zprofile) under the
// user's HOME directory.
func TestIntegrationInitPathSetupUnix(t *testing.T) {
	requireCI(t)
	binary := buildCJV(t)
	cjvHome := t.TempDir()
	fakeHome := t.TempDir()

	// Pre-create shell config files so ShellConfigPaths() finds them
	for _, rc := range []string{".profile", ".bashrc", ".zshrc", ".zprofile"} {
		require.NoError(t, os.WriteFile(filepath.Join(fakeHome, rc), []byte("# existing\n"), 0o644))
	}

	stdout, stderr, err := runCJVEnv(t, binary, cjvHome,
		[]string{"HOME=" + fakeHome},
		"init", "-y", "--default-toolchain", "none")
	require.NoError(t, err, "init failed: stdout=%s stderr=%s", stdout, stderr)

	expectedBinDir := filepath.Join(cjvHome, "bin")
	marker := "# cjv (managed by cjv, do not edit)"

	for _, rc := range []string{".profile", ".bashrc", ".zshrc", ".zprofile"} {
		content, err := os.ReadFile(filepath.Join(fakeHome, rc))
		require.NoError(t, err, "failed to read %s", rc)
		s := string(content)
		assert.Contains(t, s, marker,
			"%s should contain the cjv marker block", rc)
		assert.Contains(t, s, expectedBinDir,
			"%s should contain the bin directory path", rc)
		assert.Contains(t, s, "export PATH=",
			"%s should contain PATH export", rc)
	}
}

// TestIntegrationInitPathSetupUnixIdempotent verifies that running init twice
// does not duplicate the PATH block in shell configs.
func TestIntegrationInitPathSetupUnixIdempotent(t *testing.T) {
	requireCI(t)
	binary := buildCJV(t)
	cjvHome := t.TempDir()
	fakeHome := t.TempDir()

	for _, rc := range []string{".profile", ".bashrc", ".zshrc", ".zprofile"} {
		require.NoError(t, os.WriteFile(filepath.Join(fakeHome, rc), []byte(""), 0o644))
	}

	extraEnv := []string{"HOME=" + fakeHome}

	// Run init twice
	_, _, err := runCJVEnv(t, binary, cjvHome, extraEnv,
		"init", "-y", "--default-toolchain", "none")
	require.NoError(t, err)

	_, _, err = runCJVEnv(t, binary, cjvHome, extraEnv,
		"init", "-y", "--default-toolchain", "none")
	require.NoError(t, err)

	marker := "# cjv (managed by cjv, do not edit)"
	content, err := os.ReadFile(filepath.Join(fakeHome, ".bashrc"))
	require.NoError(t, err)
	assert.Equal(t, 1, strings.Count(string(content), marker),
		".bashrc should contain the marker block exactly once")
}

// TestIntegrationInitEnvScripts verifies that cjv init writes the POSIX env
// script to CJV_HOME on non-Windows platforms.
func TestIntegrationInitEnvScripts(t *testing.T) {
	binary := buildCJV(t)
	cjvHome := t.TempDir()
	fakeHome := t.TempDir()

	stdout, stderr, err := runCJVEnv(t, binary, cjvHome,
		[]string{"HOME=" + fakeHome},
		"init", "-y", "--default-toolchain", "none", "--no-modify-path")
	require.NoError(t, err, "init failed: stdout=%s stderr=%s", stdout, stderr)

	expectedBinDir := filepath.Join(cjvHome, "bin")

	// Check POSIX env script
	envContent, err := os.ReadFile(filepath.Join(cjvHome, "env"))
	require.NoError(t, err, "env script should exist")
	assert.Contains(t, string(envContent), expectedBinDir)
	assert.Contains(t, string(envContent), "export PATH=")
}

// TestIntegrationInitNoModifyPathSkips verifies that --no-modify-path
// prevents modification of shell config files.
func TestIntegrationInitNoModifyPathSkips(t *testing.T) {
	binary := buildCJV(t)
	cjvHome := t.TempDir()
	fakeHome := t.TempDir()

	original := "# untouched\n"
	for _, rc := range []string{".profile", ".bashrc", ".zshrc", ".zprofile"} {
		require.NoError(t, os.WriteFile(filepath.Join(fakeHome, rc), []byte(original), 0o644))
	}

	stdout, stderr, err := runCJVEnv(t, binary, cjvHome,
		[]string{"HOME=" + fakeHome},
		"init", "-y", "--default-toolchain", "none", "--no-modify-path")
	require.NoError(t, err, "init failed: stdout=%s stderr=%s", stdout, stderr)

	for _, rc := range []string{".profile", ".bashrc", ".zshrc", ".zprofile"} {
		content, err := os.ReadFile(filepath.Join(fakeHome, rc))
		require.NoError(t, err)
		assert.Equal(t, original, string(content),
			"%s should be unchanged with --no-modify-path", rc)
	}

	// Env scripts should still be written regardless of --no-modify-path
	assert.FileExists(t, filepath.Join(cjvHome, "env"))
}
