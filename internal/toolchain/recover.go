package toolchain

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/fsops"
	"github.com/Zxilly/cjv/internal/fstx"
)

// RecoverHome waits for the CJV_HOME mutation lock before recovering residue.
// This excludes live staging trees and journals. It first resumes interrupted fstx transactions under toolchains/, then removes
// abandoned staging trees and restores legacy backups whose original is
// missing. A blocked recovery is returned as a *fstx.RecoveryError with its
// journal and backups left in place, and no residue is touched: it may still
// belong to the transaction whose state could not be read. Residue cleanup
// failures are logged; they never fail the caller.
func RecoverHome() error {
	return RecoverHomeContext(context.Background())
}

// RecoverHomeContext allows callers to cancel while another process owns home.
func RecoverHomeContext(ctx context.Context) error {
	lock, err := LockHome(ctx)
	if err != nil {
		return err
	}
	defer lock.Close() //nolint:errcheck // closing releases the OS lock
	if err := lock.Recover(); err != nil {
		return err
	}
	return lock.MigrateLegacy()
}

// Recover repairs interrupted changes while the caller holds the home lock.
func (lock *HomeLock) Recover() error {
	tcDir := lock.tcDir
	if err := fstx.Recover(tcDir); err != nil {
		return err
	}
	entries, err := os.ReadDir(tcDir)
	if err != nil {
		return nil
	}
	for _, e := range entries {
		name := e.Name()
		fullPath := filepath.Join(tcDir, name)
		if strings.HasSuffix(name, config.StagingSuffix) {
			if err := fsops.RemoveAllRetry(fullPath); err != nil {
				slog.Warn("failed to clean up staging directory", "name", name, "error", err)
			}
		} else if originalName, ok := strings.CutSuffix(name, config.BackupSuffix); ok {
			// Legacy .old backups have no commit marker. Restore only when the
			// original is missing; a present original never proves commitment.
			if !filepath.IsLocal(originalName) || originalName == "." || strings.HasPrefix(originalName, config.TxTempPrefix) {
				continue
			}
			originalPath := filepath.Join(tcDir, originalName)
			if _, err := os.Lstat(originalPath); errors.Is(err, os.ErrNotExist) {
				if err := fsops.RenameRetry(fullPath, originalPath); err != nil {
					slog.Warn("failed to restore legacy toolchain backup", "path", fullPath, "error", err)
				}
			} else {
				slog.Warn("legacy toolchain backup retained", "path", fullPath, "error", err)
			}
		}
	}
	for _, subdir := range []string{config.DocsSubdir, config.StdxSubdir} {
		root := filepath.Join(filepath.Dir(tcDir), subdir)
		entries, err := os.ReadDir(root)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return err
		}
		for _, entry := range entries {
			if strings.HasSuffix(entry.Name(), config.StagingSuffix) {
				if err := fsops.RemoveAllRetry(filepath.Join(root, entry.Name())); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
