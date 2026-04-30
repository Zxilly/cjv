package target

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"  OHOS_arm32  ", "ohos-arm32"},
		{"Android", "android"},
		{"ios", "ios"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := Normalize(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNormalizeRejectsInvalidTargets(t *testing.T) {
	for _, input := range []string{"ohos/arm32", "ohos\\arm32", "linux-x64-ohos", "win32-x64", "ohos-arm64", "ohos-x64", "ohos--arm32"} {
		_, err := Normalize(input)
		assert.Error(t, err, "should reject %q", input)
	}
}

func TestNormalizeListSplitsCommaSeparatedTargets(t *testing.T) {
	got, err := NormalizeList([]string{"ohos, android", "OHOS_arm32"})
	require.NoError(t, err)
	assert.Equal(t, []string{"ohos", "android", "ohos-arm32"}, got)
}

func TestNormalizeListRejectsEmptyTargetEntries(t *testing.T) {
	_, err := NormalizeList([]string{"ohos,,android"})
	assert.Error(t, err)
}

func TestHostKey(t *testing.T) {
	tests := []struct {
		goos, goarch string
		want         string
	}{
		{"windows", "amd64", "win32-x64"},
		{"darwin", "arm64", "darwin-arm64"},
		{"darwin", "amd64", "darwin-x64"},
		{"linux", "arm64", "linux-arm64"},
		{"linux", "amd64", "linux-x64"},
	}
	for _, tt := range tests {
		t.Run(tt.goos+"-"+tt.goarch, func(t *testing.T) {
			got, err := HostKey(tt.goos, tt.goarch)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestToolchainKey(t *testing.T) {
	key, err := ToolchainKey("linux-x64", "")
	require.NoError(t, err)
	assert.Equal(t, "linux-x64", key)

	key, err = ToolchainKey("linux-x64", "ohos")
	require.NoError(t, err)
	assert.Equal(t, "linux-x64-ohos", key)
}

func TestParseToolchainKey(t *testing.T) {
	key, err := ParseToolchainKey("linux-x64-ohos-arm32")
	require.NoError(t, err)
	assert.Equal(t, "linux-x64", key.HostKey)
	assert.Equal(t, "ohos-arm32", key.Target)

	native, err := ParseToolchainKey("darwin-arm64")
	require.NoError(t, err)
	assert.Equal(t, "darwin-arm64", native.HostKey)
	assert.Empty(t, native.Target)
}

func TestParseToolchainKeyRejectsMalformedTargetSuffix(t *testing.T) {
	for _, key := range []string{"linux-x64-", "linux-x64-ohos--arm32"} {
		_, err := ParseToolchainKey(key)
		assert.Error(t, err, "should reject %q", key)
	}
}

func TestSplitVariantSuffixRequiresValidToolchainKey(t *testing.T) {
	version, platformKey := SplitVariantSuffix("1.1.0-beta-linux-x64-ohos")
	assert.Equal(t, "1.1.0-beta", version)
	assert.Equal(t, "linux-x64-ohos", platformKey)

	version, platformKey = SplitVariantSuffix("1.1.0-beta-linux-x64ohos")
	assert.Equal(t, "1.1.0-beta-linux-x64ohos", version)
	assert.Empty(t, platformKey)
}
