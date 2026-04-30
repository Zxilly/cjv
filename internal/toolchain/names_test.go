package toolchain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseToolchainName(t *testing.T) {
	tests := []struct {
		input   string
		channel Channel
		version string
	}{
		{"lts", LTS, ""},
		{"sts", STS, ""},
		{"nightly", Nightly, ""},
		{"lts-1.0.5", LTS, "1.0.5"},
		{"sts-1.1.0-beta.23", STS, "1.1.0-beta.23"},
		{"nightly-1.1.0-alpha.20260306010001", Nightly, "1.1.0-alpha.20260306010001"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			name, err := ParseToolchainName(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.channel, name.Channel)
			assert.Equal(t, tt.version, name.Version)
		})
	}
}

func TestParseToolchainNameInvalid(t *testing.T) {
	_, err := ParseToolchainName("")
	assert.Error(t, err)

	_, err = ParseToolchainName("+lts")
	assert.Error(t, err)

	_, err = ParseToolchainName("foo/bar")
	assert.Error(t, err)
}

func TestParseCustomToolchainName(t *testing.T) {
	// Custom/linked toolchain names that don't match any channel
	name, err := ParseToolchainName("my-sdk")
	require.NoError(t, err)
	assert.True(t, name.IsCustom())
	assert.Equal(t, "my-sdk", name.Custom)
	assert.Equal(t, "my-sdk", name.String())

	name, err = ParseToolchainName("unknown-1.0.0")
	require.NoError(t, err)
	assert.True(t, name.IsCustom())
	assert.Equal(t, "unknown-1.0.0", name.Custom)
}

func TestToolchainNameString(t *testing.T) {
	n := ToolchainName{Channel: LTS, Version: "1.0.5"}
	assert.Equal(t, "lts-1.0.5", n.String())

	n2 := ToolchainName{Channel: Nightly, Version: ""}
	assert.Equal(t, "nightly", n2.String())

	n3 := ToolchainName{Channel: STS, Version: "1.1.0", PlatformKey: "linux-x64-ohos"}
	assert.Equal(t, "sts-1.1.0-linux-x64-ohos", n3.String())
}

func TestIsChannelOnly(t *testing.T) {
	n := ToolchainName{Channel: LTS, Version: ""}
	assert.True(t, n.IsChannelOnly())

	n2 := ToolchainName{Channel: LTS, Version: "1.0.5"}
	assert.False(t, n2.IsChannelOnly())
}

func TestParseVersionOnly(t *testing.T) {
	// Bare version number (no channel prefix) returns UnknownChannel
	name, err := ParseToolchainName("1.0.5")
	require.NoError(t, err)
	assert.Equal(t, UnknownChannel, name.Channel)
	assert.Equal(t, "1.0.5", name.Version)
}

func TestParseToolchainTargetVariantName(t *testing.T) {
	name, err := ParseToolchainName("sts-1.1.0-beta.23-linux-x64-ohos")
	require.NoError(t, err)
	assert.Equal(t, STS, name.Channel)
	assert.Equal(t, "1.1.0-beta.23", name.Version)
	assert.Equal(t, "linux-x64-ohos", name.PlatformKey)
	assert.False(t, name.IsChannelOnly())
}

// --- Tests merged from parse_channel_test.go ---

// Tests for ParseChannel -- parses user-supplied channel names.
// Users type these in commands ("cjv install lts") and config files.

func TestParseChannel_CaseInsensitiveRecognition(t *testing.T) {
	// Channel names in config files and CLI args can be any case.
	cases := []struct {
		input string
		want  Channel
	}{
		{"lts", LTS}, {"LTS", LTS}, {"Lts", LTS},
		{"sts", STS}, {"STS", STS}, {"Sts", STS},
		{"nightly", Nightly}, {"NIGHTLY", Nightly}, {"Nightly", Nightly},
	}
	for _, tt := range cases {
		ch, ok := ParseChannel(tt.input)
		assert.True(t, ok, "should accept %q", tt.input)
		assert.Equal(t, tt.want, ch, "wrong channel for %q", tt.input)
	}
}

func TestParseChannel_RejectsInvalidNames(t *testing.T) {
	// Invalid names must return false so the caller can show a clear error
	// instead of silently using UnknownChannel.
	invalid := []string{"", "stable", "beta", "release", "lts-", "LTS "}
	for _, input := range invalid {
		_, ok := ParseChannel(input)
		assert.False(t, ok, "should reject %q", input)
	}
}

func TestParseChannel_ReturnsUnknownChannelOnFailure(t *testing.T) {
	// The Channel value on failure must be UnknownChannel, not some
	// valid channel -- callers may check the bool OR the Channel value.
	ch, ok := ParseChannel("bogus")
	assert.False(t, ok)
	assert.Equal(t, UnknownChannel, ch)
}
