package proxy

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractToolName(t *testing.T) {
	assert.Equal(t, "cjc", ExtractToolName("/home/user/.cjv/bin/cjc"))
	assert.Equal(t, "cjpm", ExtractToolName("/usr/local/bin/cjpm"))
	if runtime.GOOS == "windows" {
		assert.Equal(t, "cjc", ExtractToolName("C:\\Users\\user\\.cjv\\bin\\cjc.exe"))
		assert.Equal(t, "CJC", ExtractToolName("C:\\Users\\user\\.cjv\\bin\\CJC.EXE"))
	}
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

func TestGetRecursionCount_ValidInteger(t *testing.T) {
	t.Setenv("CJV_RECURSION_COUNT", "5")
	assert.Equal(t, 5, GetRecursionCount())
}

func TestGetRecursionCount_UnsetDefaultsToZero(t *testing.T) {
	t.Setenv("CJV_RECURSION_COUNT", "")
	assert.Equal(t, 0, GetRecursionCount())
}

func TestGetRecursionCount_InvalidStringFailsSafe(t *testing.T) {
	t.Setenv("CJV_RECURSION_COUNT", "not-a-number")
	// cjv only ever writes a valid integer, so a non-numeric value means the
	// counter was corrupted externally. Fail safe to the recursion limit so the
	// loop guard trips, rather than silently resetting to 0 and defeating it.
	assert.Equal(t, maxRecursion, GetRecursionCount())
}

func TestGetRecursionCount_NegativeClampedToZero(t *testing.T) {
	t.Setenv("CJV_RECURSION_COUNT", "-3")
	assert.Equal(t, 0, GetRecursionCount())
}
