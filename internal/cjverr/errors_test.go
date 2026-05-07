package cjverr

import (
	"testing"

	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/stretchr/testify/assert"
)

// Tests that all Error() methods produce proper human-readable messages.
// Each Error() method calls i18n.T() with a specific messageID. If the
// messageID is wrong or missing from en.toml, i18n.T() falls back to
// returning the raw messageID string. These tests catch that by verifying
// the returned message is NOT the raw ID and contains the expected data.

func TestAllErrorTypes_ProduceReadableMessages(t *testing.T) {
	i18n.Init("en")

	tests := []struct {
		name     string // also the expected messageID — if Error() returns this, the key is wrong
		err      error
		contains []string // template data that must appear in the rendered message
	}{
		{
			"ToolchainNotInstalled",
			&ToolchainNotInstalledError{Name: "lts-1.0.5"},
			[]string{"lts-1.0.5"},
		},
		{
			"ToolchainAlreadyInstalled",
			&ToolchainAlreadyInstalledError{Name: "sts-2.0.0"},
			[]string{"sts-2.0.0"},
		},
		{
			"VersionNotFound",
			&VersionNotFoundError{Version: "99.99.99"},
			[]string{"99.99.99"},
		},
		{
			"VersionNotAvailable",
			&VersionNotAvailableError{Version: "1.0.0", Platform: "linux-arm64"},
			[]string{"1.0.0", "linux-arm64"},
		},
		{
			"NoToolchainConfigured",
			&NoToolchainConfiguredError{},
			nil, // no template data, just verify non-empty
		},
		{
			"UnknownTool",
			&UnknownToolError{Name: "fakecmd"},
			[]string{"fakecmd"},
		},
		{
			"ToolNotInToolchain",
			&ToolNotInToolchainError{Tool: "cjc", Path: "/opt/sdk/lts-1.0.5"},
			[]string{"cjc", "/opt/sdk/lts-1.0.5"},
		},
		{
			"ChecksumMismatch",
			&ChecksumMismatchError{Expected: "aabbcc", Actual: "ddeeff"},
			[]string{"aabbcc", "ddeeff"},
		},
		{
			"UnsupportedPlatform",
			&UnsupportedPlatformError{OS: "plan9", Arch: "mips"},
			[]string{"plan9", "mips"},
		},
		{
			"RecursionLimitExceeded",
			&RecursionLimitError{Max: 20},
			[]string{"20"}, // int must be converted to string
		},
		{
			"UnknownChannel",
			&UnknownChannelError{Channel: "beta"},
			[]string{"beta"},
		},
		{
			"GitCodeAPIKeyRequired",
			&GitCodeAPIKeyRequiredError{},
			nil,
		},
		{
			"UnknownComponent",
			&UnknownComponentError{Name: "extra-docs"},
			[]string{"extra-docs"},
		},
		{
			"ComponentNotInstalled",
			&ComponentNotInstalledError{Toolchain: "lts-1.0.5", Component: "docs"},
			[]string{"lts-1.0.5", "docs"},
		},
		{
			"ComponentAlreadyInstalled",
			&ComponentAlreadyInstalledError{Toolchain: "lts-1.0.5", Component: "stdx"},
			[]string{"lts-1.0.5", "stdx"},
		},
		{
			"ComponentNotAvailableForChannel",
			&ComponentNotAvailableForChannelError{Component: "docs", Channel: "nightly"},
			[]string{"docs", "nightly"},
		},
		{
			"ComponentRequiresHost",
			&ComponentRequiresHostError{Component: "docs"},
			[]string{"docs"},
		},
		{
			"DocsNotInstalled",
			&DocsNotInstalledError{Toolchain: "nightly-202501010000"},
			[]string{"nightly-202501010000"},
		},
		{
			"DocsTopicNotFound",
			&DocsTopicNotFoundError{Toolchain: "nightly-202501010000", Topic: "stdx", MissingComponent: "stdx-docs"},
			[]string{"nightly-202501010000", "stdx", "stdx-docs"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := tt.err.Error()
			assert.NotEqual(t, tt.name, msg,
				"Error() returned raw messageID %q — i18n key is likely wrong or missing", tt.name)
			assert.NotEmpty(t, msg)
			for _, s := range tt.contains {
				assert.Contains(t, msg, s,
					"message should contain template data %q", s)
			}
		})
	}
}

func TestExitCodeErrorMessage(t *testing.T) {
	assert.Equal(t, "process exited with code 42", (&ExitCodeError{Code: 42}).Error())
}

func TestDocsTopicNotFoundWithoutMissingComponentOmitsInstallHint(t *testing.T) {
	i18n.Init("en")

	msg := (&DocsTopicNotFoundError{Toolchain: "lts-1.0.5", Topic: "missing"}).Error()

	assert.Contains(t, msg, "missing")
	assert.NotContains(t, msg, "component add")
}
