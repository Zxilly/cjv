package selfupdate

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

var (
	openUpdateFile   = os.OpenFile
	renameUpdateFile = os.Rename
	removeUpdateFile = os.Remove
)

// applyExecutableUpdate replaces targetPath using the same two-rename scheme
// used by go-selfupdate: write .new, move the running target to .old, then
// promote .new. A running executable can be renamed on Windows even though it
// cannot be removed until the old process exits.
func applyExecutableUpdate(update io.Reader, targetPath string, mode os.FileMode) error {
	if mode == 0 {
		mode = 0o755
	}
	newBytes, err := io.ReadAll(update)
	if err != nil {
		return err
	}

	dir := filepath.Dir(targetPath)
	name := filepath.Base(targetPath)
	newPath := filepath.Join(dir, "."+name+".new")
	oldPath := filepath.Join(dir, "."+name+".old")

	f, err := openUpdateFile(newPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer f.Close() //nolint:errcheck // explicit close below handles the actionable error
	if _, err := f.Write(newBytes); err != nil {
		return fmt.Errorf("write replacement executable: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close replacement executable: %w", err)
	}

	_ = removeUpdateFile(oldPath)
	if err := renameUpdateFile(targetPath, oldPath); err != nil {
		return fmt.Errorf("backup current executable: %w", err)
	}
	if err := renameUpdateFile(newPath, targetPath); err != nil {
		promotionErr := fmt.Errorf("promote replacement executable: %w", err)
		if rollbackErr := renameUpdateFile(oldPath, targetPath); rollbackErr != nil {
			return errors.Join(promotionErr, fmt.Errorf("restore current executable: %w", rollbackErr))
		}
		return promotionErr
	}

	if err := removeUpdateFile(oldPath); err != nil {
		_ = hideUpdateFile(oldPath)
	}
	return nil
}
