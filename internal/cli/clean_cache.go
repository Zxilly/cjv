package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/utils"
	"github.com/spf13/cobra"
)

var cleanCacheCmd = &cobra.Command{
	Use:   "clean-cache",
	Short: "Remove cached downloads",
	RunE: func(cmd *cobra.Command, args []string) error {
		removed := cleanDownloadCache()
		if removed == 0 {
			fmt.Println(i18n.T("CacheAlreadyClean", nil))
			return nil
		}
		fmt.Println(i18n.T("CacheCleanedCount", i18n.MsgData{"Count": strconv.Itoa(removed)}))
		return nil
	},
}

// cleanDownloadCache removes all entries from the downloads directory.
func cleanDownloadCache() int {
	dir, err := config.DownloadsDir()
	if err != nil {
		return 0
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			slog.Warn("failed to read download cache", "dir", dir, "error", err)
		}
		return 0
	}
	removed := 0
	for _, e := range entries {
		if err := utils.RemoveAllRetry(filepath.Join(dir, e.Name())); err == nil {
			removed++
		}
	}
	return removed
}
