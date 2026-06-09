package settings

import (
	"fmt"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/spf13/cobra"
)

var setHomeCmd = &cobra.Command{
	Use:   "home <path>",
	Short: i18n.T("SetHomeShort", nil),
	Long:  i18n.T("SetHomeLong", nil),
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		val := args[0]
		var stored string
		if val != "" {
			abs, err := filepath.Abs(val)
			if err != nil {
				return fmt.Errorf("invalid path %q: %w", val, err)
			}
			stored = abs
		}

		display := stored
		if display == "" {
			display = "(unset)"
		}
		return updateSetting("home", display, func(s *config.Settings) bool {
			if s.Home == stored {
				return false
			}
			s.Home = stored
			return true
		})
	},
}
