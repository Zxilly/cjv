package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Tests for TP -- plural translation. Used for messages like
// "1 toolchain installed" vs "3 toolchains installed".

func TestTP_SingularForm(t *testing.T) {
	Init("en")
	msg := TP("InstalledToolchains", nil, 1)
	assert.NotEqual(t, "InstalledToolchains", msg,
		"should resolve, not return raw message ID")
}

func TestTP_PluralForm(t *testing.T) {
	Init("en")
	msg := TP("InstalledToolchains", nil, 5)
	assert.NotEqual(t, "InstalledToolchains", msg)
}

func TestTP_ZeroCount(t *testing.T) {
	Init("en")
	msg := TP("InstalledToolchains", nil, 0)
	assert.NotEqual(t, "InstalledToolchains", msg)
}

func TestTP_MissingID(t *testing.T) {
	Init("en")
	msg := TP("NonExistentPluralMsg", nil, 2)
	assert.Equal(t, "NonExistentPluralMsg", msg,
		"missing ID should fall back to message ID itself")
}

func TestTP_ChinesePlural(t *testing.T) {
	Init("zh-CN")
	msg := TP("InstalledToolchains", nil, 3)
	assert.NotEqual(t, "InstalledToolchains", msg,
		"Chinese plural form should resolve")
}

// Tests for detectLanguage -- determines which locale to use.

func TestDetectLanguage_RespectsEnvVar(t *testing.T) {
	t.Setenv("CJV_LANG", "ja")

	assert.Equal(t, "ja", detectLanguage())
}

func TestDetectLanguage_FallsBackWhenUnset(t *testing.T) {
	t.Setenv("CJV_LANG", "")

	lang := detectLanguage()
	assert.NotEmpty(t, lang, "should return system locale or 'en' fallback")
}

// Tests for Init("") -- auto-detect language path.

func TestInit_EmptyStringAutoDetects(t *testing.T) {
	Init("")
	// After auto-detect, translations should still work
	msg := T("ToolchainInstalled", MsgData{"Name": "test"})
	assert.Contains(t, msg, "test")
}
