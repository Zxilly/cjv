// Package settingscmd implements the commands that edit user settings:
// default, override, and set.
package settingscmd

import (
	"io"

	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/spf13/cobra"
)

// Register adds default, override, and set to root.
func Register(root *cobra.Command, ctx *clictx.Context) {
	root.AddCommand(newSetCommand(ctx), newDefaultCommand(ctx), newOverrideCommand(ctx))
}

type settingResult struct {
	Key     string `json:"key"`
	Value   any    `json:"value"`
	Changed bool   `json:"changed"`
	display string
}

func (r settingResult) Text() string {
	if !r.Changed {
		return ""
	}
	return i18n.T("SettingUpdated", i18n.MsgData{"Key": r.Key, "Value": r.display})
}

// updateSetting persists the user's explicit choice and prints confirmation
// when it changes the user file, even if its effective value was inherited.
func updateSetting(ctx *clictx.Context, w io.Writer, key, displayValue string, value any, update config.SettingsUpdate) error {
	sf, err := config.DefaultSettingsFile()
	if err != nil {
		return err
	}
	changed, err := sf.Update(update)
	if err != nil {
		return err
	}
	return ctx.Output.RenderTo(w, settingResult{Key: key, Value: value, Changed: changed, display: displayValue})
}
