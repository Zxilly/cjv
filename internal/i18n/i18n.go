package i18n

import (
	"os"
	"sync"

	"github.com/BurntSushi/toml"
	golocale "github.com/jeandeaual/go-locale"
	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

var (
	localizer   *i18n.Localizer
	localizerMu sync.RWMutex
)

// Init initializes i18n. Pass "" for lang to auto-detect.
func Init(lang string) {
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("toml", toml.Unmarshal)

	_, _ = bundle.LoadMessageFileFS(localeFS, "locales/en.toml")
	_, _ = bundle.LoadMessageFileFS(localeFS, "locales/zh-CN.toml")

	if lang == "" {
		lang = detectLanguage()
	}

	localizerMu.Lock()
	localizer = i18n.NewLocalizer(bundle, lang, "en")
	localizerMu.Unlock()
}

func ensureInit() {
	localizerMu.RLock()
	initialized := localizer != nil
	localizerMu.RUnlock()
	if !initialized {
		Init("")
	}
}

func getLocalizer() *i18n.Localizer {
	localizerMu.RLock()
	defer localizerMu.RUnlock()
	return localizer
}

// MsgData holds template data for i18n messages. All values are strings.
type MsgData map[string]string

// T translates a message. data may be nil.
func T(messageID string, data MsgData) string {
	ensureInit()

	cfg := &i18n.LocalizeConfig{MessageID: messageID}
	if data != nil {
		cfg.TemplateData = map[string]string(data)
	}
	msg, err := getLocalizer().Localize(cfg)
	if err != nil {
		return messageID // fallback
	}
	return msg
}

// TP translates a pluralized message.
func TP(messageID string, data MsgData, count int) string {
	ensureInit()

	cfg := &i18n.LocalizeConfig{
		MessageID:   messageID,
		PluralCount: count,
	}
	if data != nil {
		cfg.TemplateData = map[string]string(data)
	}
	msg, err := getLocalizer().Localize(cfg)
	if err != nil {
		return messageID
	}
	return msg
}

func detectLanguage() string {
	if lang := os.Getenv("CJV_LANG"); lang != "" {
		return lang
	}

	if lang, err := golocale.GetLocale(); err == nil && lang != "" {
		return lang
	}

	return "en"
}
