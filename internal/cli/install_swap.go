package cli

import (
	"errors"
	"fmt"

	"github.com/Zxilly/cjv/internal/fstx"
)

func swapInstalledToolchain(stagingDir, destDir string, isReinstall bool, afterSwap func() error) (err error) {
	tx, txErr := fstx.NewTransaction(destDir)
	if txErr != nil {
		return fmt.Errorf("failed to begin install transaction: %w", txErr)
	}
	committed := false
	defer func() {
		if committed {
			return
		}
		// A rollback that itself fails (e.g. a sharing violation persisting past
		// the retries on Windows) can leave the destination missing; surface
		// that alongside the original error instead of swallowing it, so the
		// user is not told only about afterSwap while the toolchain silently
		// disappeared.
		if rbErr := tx.Rollback(); rbErr != nil {
			err = errors.Join(err, fmt.Errorf("rollback after failed install also failed: %w", rbErr))
		}
	}()

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

	if err := tx.Commit(); err != nil {
		return err
	}
	committed = true
	return nil
}
