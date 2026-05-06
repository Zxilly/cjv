package i18n

import (
	"testing"

	"github.com/BurntSushi/toml"
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

// TestLocaleKeysSynced ensures every translation key declared in en.toml has
// a counterpart in zh-CN.toml (and vice versa). Without this it's easy to add
// a new feature in one language and silently fall back to the message ID in
// the other.
func TestLocaleKeysSynced(t *testing.T) {
	en := loadLocaleKeys(t, "locales/en.toml")
	zh := loadLocaleKeys(t, "locales/zh-CN.toml")

	for k := range en {
		assert.Contains(t, zh, k, "key %q present in en.toml but missing in zh-CN.toml", k)
	}
	for k := range zh {
		assert.Contains(t, en, k, "key %q present in zh-CN.toml but missing in en.toml", k)
	}
}

func loadLocaleKeys(t *testing.T, path string) map[string]struct{} {
	t.Helper()
	data, err := localeFS.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	keys := make(map[string]struct{}, len(raw))
	for k := range raw {
		keys[k] = struct{}{}
	}
	return keys
}
