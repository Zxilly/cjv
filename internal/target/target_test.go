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

func TestHostTuple(t *testing.T) {
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
			got, err := HostTuple(tt.goos, tt.goarch)
			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestBuildTuple(t *testing.T) {
	tuple, err := BuildTuple("linux-x64", "")
	require.NoError(t, err)
	assert.Equal(t, "linux-x64", tuple)

	tuple, err = BuildTuple("linux-x64", "ohos")
	require.NoError(t, err)
	assert.Equal(t, "linux-x64-ohos", tuple)

	_, err = BuildTuple("linux-x64-ohos", "android")
	assert.Error(t, err)

	_, err = BuildTuple("unknown-host", "ohos")
	assert.Error(t, err)
}

func TestCurrentHostTupleAndHostPartOf(t *testing.T) {
	tuple, err := CurrentHostTuple("linux-amd64")
	require.NoError(t, err)
	assert.Equal(t, "linux-x64", tuple)

	_, err = CurrentHostTuple("linux")
	assert.Error(t, err)

	host, err := HostPartOf("linux-x64-ohos")
	require.NoError(t, err)
	assert.Equal(t, "linux-x64", host)
}

func TestParseTuple(t *testing.T) {
	parts, err := ParseTuple("linux-x64-ohos-arm32")
	require.NoError(t, err)
	assert.Equal(t, "linux-x64", parts.Host)
	assert.Equal(t, "ohos-arm32", parts.Environment)

	native, err := ParseTuple("darwin-arm64")
	require.NoError(t, err)
	assert.Equal(t, "darwin-arm64", native.Host)
	assert.Empty(t, native.Environment)
}

func TestParseTupleRejectsMalformedTargetSuffix(t *testing.T) {
	for _, tuple := range []string{"linux-x64-", "linux-x64-ohos--arm32"} {
		_, err := ParseTuple(tuple)
		assert.Error(t, err, "should reject %q", tuple)
	}
}

func TestSplitVariantSuffixRequiresValidTuple(t *testing.T) {
	version, tuple := SplitVariantSuffix("1.1.0-beta-linux-x64-ohos")
	assert.Equal(t, "1.1.0-beta", version)
	assert.Equal(t, "linux-x64-ohos", tuple)

	version, tuple = SplitVariantSuffix("1.1.0-beta-linux-x64ohos")
	assert.Equal(t, "1.1.0-beta-linux-x64ohos", version)
	assert.Empty(t, tuple)
}
