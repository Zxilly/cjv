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

const cleanDownloadCacheMaxPasses = 3

var cleanCacheCmd = &cobra.Command{
	Use:   "clean-cache",
	Short: "Remove cached downloads",
	RunE: func(cmd *cobra.Command, args []string) error {
		removed, err := cleanDownloadCache()
		if err != nil {
			return err
		}
		if removed == 0 {
			fmt.Println(i18n.T("CacheAlreadyClean", nil))
			return nil
		}
		fmt.Println(i18n.T("CacheCleanedCount", i18n.MsgData{"Count": strconv.Itoa(removed)}))
		return nil
	},
}

// cleanDownloadCache removes all entries from the downloads directory.
func cleanDownloadCache() (int, error) {
	dir, err := config.DownloadsDir()
	if err != nil {
		return 0, err
	}
	removed := 0

	for range cleanDownloadCacheMaxPasses {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				slog.Warn("failed to read download cache", "dir", dir, "error", err)
				return removed, fmt.Errorf("read download cache %s: %w", dir, err)
			}
			return removed, nil
		}
		if len(entries) == 0 {
			return removed, nil
		}

		var errs []error
		for _, e := range entries {
			path := filepath.Join(dir, e.Name())
			if err := removeDownloadCacheEntry(path); err == nil {
				removed++
			} else {
				errs = append(errs, fmt.Errorf("remove %s: %w", path, err))
			}
		}
		if err := errors.Join(errs...); err != nil {
			return removed, err
		}
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return removed, nil
		}
		return removed, fmt.Errorf("read download cache %s: %w", dir, err)
	}
	if len(entries) > 0 {
		return removed, fmt.Errorf("download cache still contains %d entries after cleanup", len(entries))
	}
	return removed, nil
}

func removeDownloadCacheEntry(path string) error {
	err := utils.RemoveAllRetry(path)
	if err == nil {
		return nil
	}
	if chmodErr := makeDownloadCacheEntryWritable(path); chmodErr != nil {
		return errors.Join(err, chmodErr)
	}
	if retryErr := utils.RemoveAllRetry(path); retryErr != nil {
		return errors.Join(err, retryErr)
	}
	return nil
}

func makeDownloadCacheEntryWritable(path string) error {
	return filepath.WalkDir(path, func(p string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return nil
		}
		mode := os.FileMode(0o600)
		if entry.IsDir() {
			mode = 0o700
		}
		if err := os.Chmod(p, mode); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	})
}
