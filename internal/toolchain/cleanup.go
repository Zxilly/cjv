package toolchain

import (
	"log/slog"
	"os"
	"path/filepath"
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
		}
	}
}
