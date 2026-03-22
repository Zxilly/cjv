package utils

import (
	"errors"
	"os"
	"path/filepath"
)

// WriteFileAtomic writes data to a file atomically using write-to-temp + rename.
// On Windows, true atomic file replacement is not possible, so we write to a
// temp file in the same directory, sync, and rename with retry to handle
// transient locks from virus scanners.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".cjv-tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()

	if _, err := tmp.Write(data); err != nil {
		return errors.Join(err, tmp.Close(), os.Remove(tmpName))
	}
	if err := tmp.Sync(); err != nil {
		return errors.Join(err, tmp.Close(), os.Remove(tmpName))
	}
	if err := tmp.Close(); err != nil {
		return errors.Join(err, os.Remove(tmpName))
	}

	if err := RenameRetry(tmpName, path); err != nil {
		return errors.Join(err, os.Remove(tmpName))
	}
	return nil
}
