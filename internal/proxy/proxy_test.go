package proxy

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractToolName(t *testing.T) {
	assert.Equal(t, "cjc", ExtractToolName("/home/user/.cjv/bin/cjc"))
	assert.Equal(t, "cjc", ExtractToolName("C:\\Users\\user\\.cjv\\bin\\cjc.exe"))
	assert.Equal(t, "cjpm", ExtractToolName("/usr/local/bin/cjpm"))
}

func TestExtractPlusToolchain(t *testing.T) {
	args := []string{"+nightly", "main.cj"}
	tc, remaining, err := extractPlusToolchain(args)
	require.NoError(t, err)
	assert.Equal(t, "nightly", tc)
	assert.Equal(t, []string{"main.cj"}, remaining)
}

func TestExtractPlusToolchainNone(t *testing.T) {
	args := []string{"main.cj", "-o", "out"}
	tc, remaining, err := extractPlusToolchain(args)
	require.NoError(t, err)
	assert.Equal(t, "", tc)
	assert.Equal(t, args, remaining)
}

func TestExtractPlusToolchainBare(t *testing.T) {
	args := []string{"+", "main.cj"}
	_, _, err := extractPlusToolchain(args)
	assert.Error(t, err, "bare '+' should be rejected")
}

func TestCheckRecursion(t *testing.T) {
	require.NoError(t, checkRecursion(0))
	require.NoError(t, checkRecursion(19))
	assert.Error(t, checkRecursion(20))
}

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

func TestShouldAutoInstall_RespectsExplicitSetting(t *testing.T) {
	s := config.DefaultSettings()

	s.AutoInstall = true
	assert.True(t, shouldAutoInstall(&s), "should auto-install when explicitly enabled")

	s.AutoInstall = false
	assert.False(t, shouldAutoInstall(&s), "should not auto-install when explicitly disabled")
}

func TestShouldAutoInstall_NilSettingsReturnsFalse(t *testing.T) {
	// When settings is nil (e.g., settings loading failed), shouldAutoInstall
	// returns false as a safe default. The caller (Run) is responsible for
	// loading settings upfront.
	assert.False(t, shouldAutoInstall(nil),
		"should return false when settings is nil")
}

func TestGetRecursionCount_ValidInteger(t *testing.T) {
	t.Setenv("CJV_RECURSION_COUNT", "5")
	assert.Equal(t, 5, GetRecursionCount())
}

func TestGetRecursionCount_UnsetDefaultsToZero(t *testing.T) {
	t.Setenv("CJV_RECURSION_COUNT", "")
	assert.Equal(t, 0, GetRecursionCount())
}

func TestGetRecursionCount_InvalidStringDefaultsToZero(t *testing.T) {
	t.Setenv("CJV_RECURSION_COUNT", "not-a-number")
	assert.Equal(t, 0, GetRecursionCount())
}

func TestGetRecursionCount_NegativeClampedToZero(t *testing.T) {
	t.Setenv("CJV_RECURSION_COUNT", "-3")
	assert.Equal(t, 0, GetRecursionCount())
}
