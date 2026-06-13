package cli

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/utils"
)

// purgeDownloadsDir wipes leftover entries from the downloads staging area.
// Steady state is empty (each install removes its archive on success), so
// this is a sweep for whatever a crashed/aborted run left behind. Called by
// `cjv update` as an end-of-update cleanup step.
const purgeDownloadsMaxPasses = 3

func purgeDownloadsDir() (int, error) {
	dir, err := config.DownloadsDir()
	if err != nil {
		return 0, err
	}
	removed := 0

	for range purgeDownloadsMaxPasses {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				slog.Warn("failed to read downloads dir", "dir", dir, "error", err)
				return removed, fmt.Errorf("read downloads dir %s: %w", dir, err)
			}
			return removed, nil
		}
		if len(entries) == 0 {
			return removed, nil
		}

		var errs []error
		for _, e := range entries {
			path := filepath.Join(dir, e.Name())
			if err := removePurgeEntry(path); err == nil {
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
		return removed, fmt.Errorf("read downloads dir %s: %w", dir, err)
	}
	if len(entries) > 0 {
		return removed, fmt.Errorf("downloads dir still contains %d entries after cleanup", len(entries))
	}
	return removed, nil
}

func removePurgeEntry(path string) error {
	err := utils.RemoveAllRetry(path)
	if err == nil {
		return nil
	}
	if chmodErr := makePurgeEntryWritable(path); chmodErr != nil {
		return errors.Join(err, chmodErr)
	}
	if retryErr := utils.RemoveAllRetry(path); retryErr != nil {
		return errors.Join(err, retryErr)
	}
	return nil
}

func makePurgeEntryWritable(path string) error {
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
