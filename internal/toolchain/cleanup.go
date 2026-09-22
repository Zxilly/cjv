package toolchain

import (
	"errors"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/fstx"
	"github.com/Zxilly/cjv/internal/utils"
)

// CleanupStagingDirs first recovers interrupted transactions, then removes
// abandoned staging trees. If recovery is blocked, keep all staging data: it
// may still be part of a transaction whose state could not be read.
func CleanupStagingDirs() {
	tcDir, err := config.ToolchainsDir()
	if err != nil {
		return
	}
	if err := fstx.Recover(tcDir); err != nil {
		slog.Warn("failed to recover install transaction", "error", err)
		return
	}
	entries, err := os.ReadDir(tcDir)
	if err != nil {
		return
	}
	for _, e := range entries {
		name := e.Name()
		fullPath := filepath.Join(tcDir, name)
		if strings.HasSuffix(name, StagingSuffix) {
			if err := utils.RemoveAllRetry(fullPath); err != nil {
				slog.Warn("failed to clean up staging directory", "name", name, "error", err)
			}
		} else if originalName, ok := strings.CutSuffix(name, BackupSuffix); ok {
			// Legacy .old backups have no commit marker. Restore only when the
			// original is missing; a present original never proves commitment.
			if !filepath.IsLocal(originalName) || originalName == "." || strings.HasPrefix(originalName, FstxTempPrefix) {
				continue
			}
			originalPath := filepath.Join(tcDir, originalName)
			if _, err := os.Lstat(originalPath); errors.Is(err, os.ErrNotExist) {
				if err := utils.RenameRetry(fullPath, originalPath); err != nil {
					slog.Warn("failed to restore legacy toolchain backup", "path", fullPath, "error", err)
				}
			} else {
				slog.Warn("legacy toolchain backup retained", "path", fullPath, "error", err)
			}
		}
	}
}
