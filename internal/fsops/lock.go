package fsops

import (
	"context"
	"fmt"
	"os"
	"time"
)

// LockFile acquires an exclusive interprocess lock. Closing the returned file
// releases it, including when the process exits without running its defers.
// The lock file must stay in place: unlinking it could let another process
// lock a different inode while an existing holder is still working.
func LockFile(ctx context.Context, path string) (*os.File, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o600)
	if err != nil {
		return nil, err
	}
	for {
		if err := ctx.Err(); err != nil {
			_ = file.Close() //nolint:errcheck // preserve the cancellation error
			return nil, err
		}
		locked, err := tryLockFile(file)
		if err != nil {
			_ = file.Close() //nolint:errcheck // preserve the locking error
			return nil, fmt.Errorf("lock %s: %w", path, err)
		}
		if locked {
			return file, nil
		}
		timer := time.NewTimer(25 * time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
		case <-timer.C:
		}
	}
}
