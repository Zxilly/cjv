package settings

import (
	"fmt"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
)

// LoadSettings loads the default settings file and its contents.
func LoadSettings() (*config.SettingsFile, *config.Settings, error) {
	sf, err := config.DefaultSettingsFile()
	if err != nil {
		return nil, nil, err
	}
	s, err := sf.Load()
	if err != nil {
		return nil, nil, err
	}
	return sf, s, nil
}

// updateSetting loads settings, applies mutate, saves if changed, and prints confirmation.
// mutate should return true if the setting was changed.
func updateSetting(key, displayValue string, mutate func(*config.Settings) bool) error {
	sf, settings, err := LoadSettings()
	if err != nil {
		return err
	}
	if !mutate(settings) {
		return nil
	}
	if err := sf.Save(settings); err != nil {
		return err
	}
	fmt.Println(i18n.T("SettingUpdated", i18n.MsgData{
		"Key":   key,
		"Value": displayValue,
	}))
	return nil
}
