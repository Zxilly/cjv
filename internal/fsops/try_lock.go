package fsops

import (
	"fmt"
	"os"
)

// TryLockFile attempts the same exclusive process lock as LockFile, without
// waiting. A busy lock returns (nil, false, nil). The caller must close a
// successfully locked file and must never unlink the lock file.
func TryLockFile(path string) (*os.File, bool, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, false, err
	}
	locked, err := tryLockFile(file)
	if err != nil || !locked {
		_ = file.Close()
		if err != nil {
			err = fmt.Errorf("lock %s: %w", path, err)
		}
		return nil, false, err
	}
	return file, true, nil
}
