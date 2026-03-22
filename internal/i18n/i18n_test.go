package i18n

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultLanguageEnglish(t *testing.T) {
	Init("en")
	msg := T("ToolchainInstalled", MsgData{"Name": "lts-1.0.5"})
	assert.Contains(t, msg, "lts-1.0.5")
	assert.Contains(t, msg, "installed")
}

func TestChineseTranslation(t *testing.T) {
	Init("zh-CN")
	msg := T("ToolchainInstalled", MsgData{"Name": "lts-1.0.5"})
	assert.Contains(t, msg, "lts-1.0.5")
	assert.Contains(t, msg, "安装成功")
}

func TestFallbackToEnglish(t *testing.T) {
	Init("fr") // unsupported language falls back to English
	msg := T("ToolchainInstalled", MsgData{"Name": "lts-1.0.5"})
	assert.Contains(t, msg, "installed")
}

func TestMissingMessageID(t *testing.T) {
	Init("en")
	msg := T("NonExistentMessage", nil)
	assert.Equal(t, "NonExistentMessage", msg) // falls back to message ID
}
