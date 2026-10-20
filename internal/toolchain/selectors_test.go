package toolchain

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/target"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionSelectorsChooseInstalledConcreteRelease(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	for _, name := range []string{"sts-1.2.1", "sts-1.2.10", "sts-1.2.11-beta.1", "nightly-1.3.0-alpha.20261004001050", "nightly-1.3.0-alpha.20261005001050"} {
		require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", name), 0755))
	}
	for _, test := range []struct{ input, actual string }{{"1.2", "sts-1.2.10"}, {"sts-1.2", "sts-1.2.10"}, {"nightly-2026-10-04", "nightly-1.3.0-alpha.20261004001050"}} {
		name, err := ParseToolchainName(test.input)
		require.NoError(t, err)
		dir, err := FindInstalled(name)
		require.NoError(t, err)
		require.Equal(t, test.actual, filepath.Base(dir))
	}
	require.False(t, MatchesVersionSelector("2026-02-30", "1.3.0-alpha.20260230000000"))
}

func TestExplicitHostSelectorsFindLegacyIdentity(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	host, err := target.CurrentHostTuple("")
	require.NoError(t, err)
	dir := filepath.Join(home, "toolchains", "sts-1.0.0")
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, WriteInstallation(dir, Installation{Release: "sts-1.0.0", Tuple: host}))
	for _, input := range []string{"sts-1.0.0-" + host, "sts-1.0-" + host, "1.0.0-" + host, "1.0-" + host} {
		name, err := ParseToolchainName(input)
		require.NoError(t, err)
		actual, err := FindInstalled(name)
		require.NoError(t, err)
		require.Equal(t, dir, actual)
	}
}

func TestCompareVersionsStandard(t *testing.T) {
	assert.Equal(t, 0, CompareVersions("1.0.0", "1.0.0"))
	assert.Equal(t, -1, CompareVersions("1.0.0", "2.0.0"))
	assert.Equal(t, 1, CompareVersions("2.0.0", "1.0.0"))
	assert.Equal(t, -1, CompareVersions("1.0.0", "1.1.0"))
	assert.Equal(t, -1, CompareVersions("1.0.0", "1.0.1"))
}

func TestCompareVersionsPreRelease(t *testing.T) {
	// Pre-release versions come before release
	assert.Equal(t, -1, CompareVersions("1.0.0-beta.1", "1.0.0"))
	assert.Equal(t, 1, CompareVersions("1.0.0", "1.0.0-beta.1"))
}

func TestCompareVersionsOneValidOneInvalid(t *testing.T) {
	// A valid semver sorts after an invalid string
	assert.Equal(t, 1, CompareVersions("1.0.0", "not-a-version"))
	assert.Equal(t, -1, CompareVersions("not-a-version", "1.0.0"))
}

func TestCompareVersionsBothInvalid(t *testing.T) {
	// Falls back to lexicographic string comparison
	assert.Equal(t, -1, CompareVersions("aaa", "bbb"))
	assert.Equal(t, 1, CompareVersions("bbb", "aaa"))
	assert.Equal(t, 0, CompareVersions("same", "same"))
}

func TestCompareVersionsPrefersReleaseOverPrerelease(t *testing.T) {
	assert.Greater(t, CompareVersions("1.8.0", "1.8.0-beta.2"), 0)
	assert.Less(t, CompareVersions("1.8.0-beta.2", "1.8.0"), 0)
}
