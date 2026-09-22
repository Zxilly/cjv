package settings

import (
	"fmt"
	"io"

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

// updateSetting persists the user's explicit choice and prints confirmation
// when it changes the user file, even if its effective value was inherited.
func updateSetting(w io.Writer, key, displayValue string, update config.SettingsUpdate) error {
	sf, err := config.DefaultSettingsFile()
	if err != nil {
		return err
	}
	changed, err := sf.Update(update)
	if err != nil {
		return err
	}
	if !changed {
		return nil
	}
	_, err = fmt.Fprintln(w, i18n.T("SettingUpdated", i18n.MsgData{
		"Key":   key,
		"Value": displayValue,
	}))
	return err
}
