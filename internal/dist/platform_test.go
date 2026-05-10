package dist

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlatformKeyFromGo(t *testing.T) {
	tests := []struct {
		goos, goarch string
		expected     string
	}{
		{"windows", "amd64", "win32-x64"},
		{"darwin", "arm64", "darwin-arm64"},
		{"darwin", "amd64", "darwin-x64"},
		{"linux", "amd64", "linux-x64"},
		{"linux", "arm64", "linux-arm64"},
	}
	for _, tt := range tests {
		t.Run(tt.goos+"-"+tt.goarch, func(t *testing.T) {
			key, err := HostTupleFromGo(tt.goos, tt.goarch)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, key)
		})
	}
}

func TestPlatformKeyUnsupported(t *testing.T) {
	_, err := HostTupleFromGo("freebsd", "amd64")
	assert.Error(t, err)
}

func TestNightlyFilename(t *testing.T) {
	tests := []struct {
		goos, goarch string
		version      string
		expected     string
	}{
		{"windows", "amd64", "1.1.0-alpha.20260306010001", "cangjie-sdk-windows-x64-1.1.0-alpha.20260306010001.zip"},
		{"darwin", "arm64", "1.1.0-alpha.20260306010001", "cangjie-sdk-mac-aarch64-1.1.0-alpha.20260306010001.tar.gz"},
		{"darwin", "amd64", "1.1.0-alpha.20260306010001", "cangjie-sdk-mac-x64-1.1.0-alpha.20260306010001.tar.gz"},
		{"linux", "amd64", "1.1.0-alpha.20260306010001", "cangjie-sdk-linux-x64-1.1.0-alpha.20260306010001.tar.gz"},
	}
	for _, tt := range tests {
		t.Run(tt.goos+"-"+tt.goarch, func(t *testing.T) {
			name, err := NightlyFilename(tt.goos, tt.goarch, tt.version)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, name)
		})
	}
}

func TestArchiveExt(t *testing.T) {
	assert.Equal(t, ".zip", ArchiveExt("windows"))
	assert.Equal(t, ".tar.gz", ArchiveExt("darwin"))
	assert.Equal(t, ".tar.gz", ArchiveExt("linux"))
}

// --- Tests merged from quick_coverage_test.go (CurrentHostTuple is in platform.go) ---

func TestCurrentPlatformKey_ReturnsValid(t *testing.T) {
	// On any supported platform, this should succeed.
	key, err := CurrentHostTuple("")
	assert.NoError(t, err)
	assert.NotEmpty(t, key)
}

func TestCurrentPlatformKeyWithTarget(t *testing.T) {
	key, err := CurrentTargetTuple("linux-amd64", "ohos")
	require.NoError(t, err)
	assert.Equal(t, "linux-x64-ohos", key)
}

func TestNightlyFilenameForPlatformWithTarget(t *testing.T) {
	name, err := NightlyFilenameForTuple("win32-x64-ohos-arm32", "1.1.0-alpha.20260429010057")
	require.NoError(t, err)
	assert.Equal(t, "cangjie-sdk-windows-x64-ohos-arm32-1.1.0-alpha.20260429010057.zip", name)
}

func TestPlatformHelpersErrorAndMappingBranches(t *testing.T) {
	_, err := CurrentHostTuple("unsupported-host")
	require.Error(t, err)

	_, err = CurrentTargetTuple("unsupported-host", "ohos")
	require.Error(t, err)

	_, err = NightlyFilename("plan9", "amd64", "1.0.0")
	require.Error(t, err)

	_, err = NightlyFilenameForTuple("unsupported-host", "1.0.0")
	require.Error(t, err)

	assert.Equal(t, "windows", NightlyGOOS("windows"))
	assert.Equal(t, "darwin", NightlyGOOS("mac"))
	assert.Equal(t, "linux", NightlyGOOS("linux"))
}
