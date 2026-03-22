package toolchain

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- ListInstalled behavioral tests ---

func TestListInstalled_ReturnsSortedNames(t *testing.T) {
	// Users running "cjv show installed" expect a consistent, sorted display.
	tmpDir := t.TempDir()
	t.Setenv("CJV_HOME", tmpDir)

	tcDir := filepath.Join(tmpDir, "toolchains")
	for _, name := range []string{"sts-2.0.0", "lts-1.0.5", "nightly-20260301"} {
		require.NoError(t, os.MkdirAll(filepath.Join(tcDir, name), 0o755))
	}

	installed, err := ListInstalled()
	require.NoError(t, err)
	assert.Equal(t, []string{"lts-1.0.5", "nightly-20260301", "sts-2.0.0"}, installed)
}

func TestListInstalled_ExcludesTemporaryDirs(t *testing.T) {
	// During install, .staging and .old directories exist temporarily.
	// They should never appear as installed toolchains to the user.
	tmpDir := t.TempDir()
	t.Setenv("CJV_HOME", tmpDir)

	tcDir := filepath.Join(tmpDir, "toolchains")
	require.NoError(t, os.MkdirAll(filepath.Join(tcDir, "lts-1.0.5"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tcDir, "lts-1.0.6.staging"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tcDir, "lts-1.0.4.old"), 0o755))

	installed, err := ListInstalled()
	require.NoError(t, err)
	assert.Equal(t, []string{"lts-1.0.5"}, installed)
}

func TestListInstalled_ExcludesRegularFiles(t *testing.T) {
	// Only directories represent installed toolchains; stray files should be ignored.
	tmpDir := t.TempDir()
	t.Setenv("CJV_HOME", tmpDir)

	tcDir := filepath.Join(tmpDir, "toolchains")
	require.NoError(t, os.MkdirAll(filepath.Join(tcDir, "lts-1.0.5"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(tcDir, "stray-file.txt"), []byte("x"), 0o644))

	installed, err := ListInstalled()
	require.NoError(t, err)
	assert.Equal(t, []string{"lts-1.0.5"}, installed)
}

func TestListInstalled_ReturnsNilBeforeFirstInstall(t *testing.T) {
	// Before the user installs any toolchain, the toolchains directory doesn't
	// exist. This should return nil (not an error), so callers can distinguish
	// "no toolchains dir" from "empty toolchains dir".
	tmpDir := t.TempDir()
	t.Setenv("CJV_HOME", tmpDir)

	installed, err := ListInstalled()
	assert.NoError(t, err)
	assert.Nil(t, installed)
}

// --- FindInstalled behavioral tests ---

func TestFindInstalled_BareVersionSearchesAllChannels(t *testing.T) {
	// User runs "cjv run +1.0.5 cjc" without specifying the channel.
	// The system should find it whether it's under lts/, sts/, or nightly/.
	tmpDir := t.TempDir()
	t.Setenv("CJV_HOME", tmpDir)

	tcDir := filepath.Join(tmpDir, "toolchains")
	require.NoError(t, os.MkdirAll(filepath.Join(tcDir, "sts-1.0.5"), 0o755))

	name := ToolchainName{Channel: UnknownChannel, Version: "1.0.5"}
	dir, err := FindInstalled(name)
	require.NoError(t, err)
	assert.Contains(t, dir, "sts-1.0.5")
}

func TestFindInstalled_DirectLookupByChannelAndVersion(t *testing.T) {
	// Exact match: "lts-1.0.5" should resolve to that specific directory.
	tmpDir := t.TempDir()
	t.Setenv("CJV_HOME", tmpDir)

	tcDir := filepath.Join(tmpDir, "toolchains")
	require.NoError(t, os.MkdirAll(filepath.Join(tcDir, "lts-1.0.5"), 0o755))

	name := ToolchainName{Channel: LTS, Version: "1.0.5"}
	dir, err := FindInstalled(name)
	require.NoError(t, err)
	assert.Contains(t, dir, "lts-1.0.5")
}

func TestFindInstalled_DirectLookupMissing(t *testing.T) {
	// Requesting a specific version that isn't installed.
	tmpDir := t.TempDir()
	t.Setenv("CJV_HOME", tmpDir)
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "toolchains"), 0o755))

	name := ToolchainName{Channel: LTS, Version: "9.9.9"}
	_, err := FindInstalled(name)
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestFindInstalled_IgnoresStagingDuringChannelSearch(t *testing.T) {
	// While a new version is being installed (lts-1.0.6.staging exists),
	// "cjv run cjc" with default=lts should pick lts-1.0.5, not the incomplete staging dir.
	tmpDir := t.TempDir()
	t.Setenv("CJV_HOME", tmpDir)

	tcDir := filepath.Join(tmpDir, "toolchains")
	require.NoError(t, os.MkdirAll(filepath.Join(tcDir, "lts-1.0.5"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tcDir, "lts-1.0.6.staging"), 0o755))

	name := ToolchainName{Channel: LTS}
	dir, err := FindInstalled(name)
	require.NoError(t, err)
	assert.Contains(t, dir, "lts-1.0.5")
}

// --- FindInstalledByName tests ---

// Tests for FindInstalledByName -- looks up custom-linked toolchains
// by their exact directory name (e.g., "my-sdk" from "cjv toolchain link").
// Unlike FindInstalled, this does no channel/version parsing.

func TestFindInstalledByName_ExactDirectoryMatch(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CJV_HOME", tmpDir)

	sdkDir := filepath.Join(tmpDir, "toolchains", "my-custom-sdk")
	require.NoError(t, os.MkdirAll(sdkDir, 0o755))

	path, err := FindInstalledByName("my-custom-sdk")
	require.NoError(t, err)
	assert.Equal(t, sdkDir, path)
}

func TestFindInstalledByName_ReturnsErrNotExist(t *testing.T) {
	// When the directory doesn't exist, callers need os.ErrNotExist
	// to distinguish "not installed" from other errors.
	tmpDir := t.TempDir()
	t.Setenv("CJV_HOME", tmpDir)
	require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "toolchains"), 0o755))

	_, err := FindInstalledByName("no-such-sdk")
	assert.ErrorIs(t, err, os.ErrNotExist)
}

func TestFindInstalledByName_WorksWithNonStandardNames(t *testing.T) {
	// Custom names don't follow channel-version format.
	tmpDir := t.TempDir()
	t.Setenv("CJV_HOME", tmpDir)

	for _, name := range []string{"cangjie-ce-0.58.2", "local-dev-build", "test"} {
		require.NoError(t, os.MkdirAll(filepath.Join(tmpDir, "toolchains", name), 0o755))

		path, err := FindInstalledByName(name)
		require.NoError(t, err, "should find %q", name)
		assert.Contains(t, path, name)
	}
}

// --- SemVer ordering and sort tests ---

func TestCompareSemVerPrefersReleaseOverPrerelease(t *testing.T) {
	assert.Greater(t, compareSemVer("1.8.0", "1.8.0-beta.2"), 0)
	assert.Less(t, compareSemVer("1.8.0-beta.2", "1.8.0"), 0)
}

func TestFindInstalledUsesSemVerOrdering(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)

	tcDir, err := config.ToolchainsDir()
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(tcDir, "sts-1.8.0-beta.2"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tcDir, "sts-1.8.0"), 0o755))

	dir, err := FindInstalled(ToolchainName{Channel: STS})
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(tcDir, "sts-1.8.0"), dir)
}
