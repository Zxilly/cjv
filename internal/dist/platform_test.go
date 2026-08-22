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

func TestPlatformHelpersErrorBranches(t *testing.T) {
	_, err := CurrentHostTuple("unsupported-host")
	require.Error(t, err)

	_, err = CurrentTargetTuple("unsupported-host", "ohos")
	require.Error(t, err)

}
