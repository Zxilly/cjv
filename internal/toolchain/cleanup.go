package toolchain

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/utils"
)

// CleanupStagingDirs removes stale .staging directories and restores
// orphaned .old backup directories from interrupted force-installs.
func CleanupStagingDirs() {
	tcDir, err := config.ToolchainsDir()
	if err != nil {
		return
	}
	entries, err := os.ReadDir(tcDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		if !IsTempDir(name) {
			continue
		}
		fullPath := filepath.Join(tcDir, name)
		if strings.HasSuffix(name, StagingSuffix) {
			if err := utils.RemoveAllRetry(fullPath); err != nil {
				slog.Warn("failed to clean up staging directory", "name", name, "error", err)
			}
		} else if originalName, ok := strings.CutSuffix(name, BackupSuffix); ok {
			// Try to restore .old backup; if rename fails (original exists), remove stale backup.
			originalPath := filepath.Join(tcDir, originalName)
			if err := utils.RenameRetry(fullPath, originalPath); err != nil {
				if err := utils.RemoveAllRetry(fullPath); err != nil {
					slog.Warn("failed to remove stale backup", "name", name, "error", err)
				}
			}
		} else if strings.HasPrefix(name, FstxTempPrefix) {
			cleanupFstxTransactionDir(tcDir, fullPath)
		}
	}
}

func cleanupFstxTransactionDir(tcDir, txDir string) {
	entries, err := os.ReadDir(txDir)
	if err != nil {
		slog.Warn("failed to read transaction backup directory", "path", txDir, "error", err)
		return
	}
	keepTxDir := false
	for _, e := range entries {
		originalName, ok := fstxBackupOriginalName(e.Name())
		if !ok {
			continue
		}

		backupPath := filepath.Join(txDir, e.Name())
		originalPath := filepath.Join(tcDir, originalName)
		if _, err := os.Lstat(originalPath); errors.Is(err, os.ErrNotExist) {
			if err := utils.RenameRetry(backupPath, originalPath); err != nil {
				slog.Warn("failed to restore transaction backup", "backup", backupPath, "original", originalPath, "error", err)
				keepTxDir = true
			}
		} else if err != nil {
			slog.Warn("failed to stat transaction backup original", "original", originalPath, "error", err)
			keepTxDir = true
		} else if err := utils.RemoveAllRetry(backupPath); err != nil {
			slog.Warn("failed to remove stale transaction backup", "backup", backupPath, "error", err)
			keepTxDir = true
		}
	}
	if keepTxDir {
		return
	}
	if err := utils.RemoveAllRetry(txDir); err != nil {
		slog.Warn("failed to remove transaction backup directory", "path", txDir, "error", err)
	}
}

func fstxBackupOriginalName(name string) (string, bool) {
	idx := strings.IndexByte(name, '-')
	if idx <= 0 || idx == len(name)-1 {
		return "", false
	}
	if _, err := strconv.Atoi(name[:idx]); err != nil {
		return "", false
	}
	return name[idx+1:], true
}
