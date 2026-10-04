package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/fsops"
)

// purgeDownloadsDir wipes leftover entries from the downloads staging area.
// Steady state is empty (each install removes its archive on success), so
// this is a sweep for whatever a crashed/aborted run left behind. Called by
// UpdateAll as its end-of-update cleanup step.
const purgeDownloadsMaxPasses = 3

func purgeDownloadsDir() (int, error) {
	return purgeDownloadsDirContext(context.Background())
}

func purgeDownloadsDirContext(ctx context.Context) (int, error) {
	dir, err := config.DownloadsDir()
	if err != nil {
		return 0, err
	}
	if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	home, err := config.Home()
	if err != nil {
		return 0, err
	}
	// A sweep must never delete a live SDK/component download or private stage.
	lock, err := fsops.LockFile(ctx, filepath.Join(home, ".install.lock"))
	if err != nil {
		return 0, err
	}
	defer lock.Close() //nolint:errcheck
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
	err := fsops.RemoveAllRetry(path)
	if err == nil {
		return nil
	}
	if chmodErr := makePurgeEntryWritable(path); chmodErr != nil {
		return errors.Join(err, chmodErr)
	}
	if retryErr := fsops.RemoveAllRetry(path); retryErr != nil {
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
