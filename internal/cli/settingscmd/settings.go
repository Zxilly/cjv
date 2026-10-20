// Package settingscmd implements the commands that edit user settings:
// default, override, and set.
package settingscmd

import (
	"fmt"
	"io"

	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/spf13/cobra"
)

// Register adds default, override, and set to root.
func Register(root *cobra.Command, ctx *clictx.Context) {
	root.AddCommand(newSetCommand(), newDefaultCommand(ctx), newOverrideCommand())
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
