package cli

import (
	"fmt"

	"github.com/Zxilly/cjv/internal/fstx"
)

func swapInstalledToolchain(stagingDir, destDir string, isReinstall bool, afterSwap func() error) error {
	tx, err := fstx.NewTransaction(destDir)
	if err != nil {
		return fmt.Errorf("failed to begin install transaction: %w", err)
	}
	defer tx.Rollback()

	if isReinstall {
		if err := tx.RemoveDir(destDir); err != nil {
			return fmt.Errorf("failed to remove existing toolchain: %w", err)
		}
	}

	if err := tx.RenameFile(stagingDir, destDir); err != nil {
		return fmt.Errorf("failed to place new toolchain: %w", err)
	}

	if err := afterSwap(); err != nil {
		return fmt.Errorf("failed to finalize installation: %w", err)
	}

	return tx.Commit()
}
