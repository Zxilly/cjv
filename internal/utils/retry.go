package utils

import (
	"errors"
	"math/rand/v2"
	"os"
	"runtime"
	"time"
)

// IsRetryableError returns true for transient filesystem errors
// (commonly caused by virus scanners or indexers on Windows).
// On Unix, permission errors are not transient and should not be retried.
func IsRetryableError(err error) bool {
	if runtime.GOOS != "windows" {
		return false
	}
	// Windows: Permission denied, sharing violation, directory not empty
	// are commonly transient due to Defender, search indexer, etc.
	return errors.Is(err, os.ErrPermission) ||
		isWindowsDirNotEmpty(err) ||
		isWindowsSharingViolation(err)
}

// RetryWithBackoff retries fn with Fibonacci backoff plus light jitter.
// It retries only when shouldRetry returns true for the error.
func RetryWithBackoff(maxAttempts int, shouldRetry func(error) bool, fn func() error) error {
	var err error
	for i := 0; i < maxAttempts; i++ {
		err = fn()
		if err == nil {
			return nil
		}
		if !shouldRetry(err) {
			return err
		}
		if i < maxAttempts-1 {
			time.Sleep(retryDelay(i))
		}
	}
	return err
}

func retryDelay(attempt int) time.Duration {
	a, b := 10*time.Millisecond, 10*time.Millisecond
	for i := 0; i < attempt; i++ {
		a, b = b, a+b
		if b > time.Second {
			b = time.Second
		}
	}

	jitter := time.Duration(rand.Int64N(int64(a/2) + 1))
	return a + jitter
}

// RemoveAllRetry removes a path with retry for transient errors.
func RemoveAllRetry(path string) error {
	return RetryWithBackoff(10, IsRetryableError, func() error {
		return os.RemoveAll(path)
	})
}

// RenameRetry renames with retry for transient errors.
func RenameRetry(oldpath, newpath string) error {
	return RetryWithBackoff(10, IsRetryableError, func() error {
		return os.Rename(oldpath, newpath)
	})
}
