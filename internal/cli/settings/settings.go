package settings

import (
	"fmt"
	"io"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
)

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
