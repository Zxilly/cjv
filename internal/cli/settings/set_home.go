package settings

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/spf13/cobra"
)

var setHomeCmd = &cobra.Command{
	Use:   "home <path>",
	Short: "Set persistent CJV_HOME path",
	Long: "Persist the cjv data directory in settings.toml. Pass an empty string to clear the override. " +
		"The CJV_HOME environment variable still takes precedence when set.",
	Args: cobra.ExactArgs(1),
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
		changed := false
		if err := updateSetting("home", display, func(s *config.Settings) bool {
			if s.Home == stored {
				return false
			}
			s.Home = stored
			changed = true
			return true
		}); err != nil {
			return err
		}
		if changed && stored != "" {
			fmt.Fprintf(os.Stderr, "\n%s\n", i18n.T("SetHomeNoMigrate", i18n.MsgData{"Path": stored}))
		}
		return nil
	},
}
