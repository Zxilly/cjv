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

// RecoverHome is the single recovery entry point for CJV_HOME. It first
// resumes interrupted fstx transactions under toolchains/, then removes
// abandoned staging trees and restores legacy backups whose original is
// missing. A blocked recovery is returned as a *fstx.RecoveryError with its
// journal and backups left in place, and no residue is touched: it may still
// belong to the transaction whose state could not be read. Residue cleanup
// failures are logged; they never fail the caller.
func RecoverHome() error {
	tcDir, err := config.ToolchainsDir()
	if err != nil {
		return err
	}
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
			if err := utils.RemoveAllRetry(fullPath); err != nil {
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
				if err := utils.RenameRetry(fullPath, originalPath); err != nil {
					slog.Warn("failed to restore legacy toolchain backup", "path", fullPath, "error", err)
				}
			} else {
				slog.Warn("legacy toolchain backup retained", "path", fullPath, "error", err)
			}
		}
	}
	return nil
}
