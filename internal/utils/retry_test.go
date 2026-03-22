package utils

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Tests for RetryWithBackoff -- the shared retry engine used by
// file operations (rename, remove) to handle transient Windows
// file locking from virus scanners.

func TestRetryWithBackoff_SucceedsImmediately(t *testing.T) {
	// Normal case: the operation works on first try, no retries needed.
	attempts := 0
	err := RetryWithBackoff(3, func(error) bool { return true }, func() error {
		attempts++
		return nil
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, attempts, "should call fn exactly once when it succeeds")
}

func TestRetryWithBackoff_RecoversFromTransientFailure(t *testing.T) {
	// Simulates a virus scanner briefly locking a file: first 2 attempts
	// fail, 3rd succeeds. The operation should ultimately succeed.
	attempts := 0
	err := RetryWithBackoff(5, func(error) bool { return true }, func() error {
		attempts++
		if attempts < 3 {
			return errors.New("file locked by another process")
		}
		return nil
	})
	assert.NoError(t, err, "should succeed after transient failures clear")
	assert.Equal(t, 3, attempts)
}

func TestRetryWithBackoff_StopsOnPermanentError(t *testing.T) {
	// When shouldRetry returns false (e.g., "file not found" -- not transient),
	// the engine should stop immediately without wasting time on more attempts.
	attempts := 0
	permanentErr := errors.New("file not found")
	err := RetryWithBackoff(10, func(error) bool { return false }, func() error {
		attempts++
		return permanentErr
	})
	assert.ErrorIs(t, err, permanentErr)
	assert.Equal(t, 1, attempts, "should not retry on permanent errors")
}

func TestRetryWithBackoff_ExhaustsAllAttempts(t *testing.T) {
	// All retries fail → returns the error from the last attempt.
	attempts := 0
	err := RetryWithBackoff(3, func(error) bool { return true }, func() error {
		attempts++
		return errors.New("still locked")
	})
	assert.Error(t, err)
	assert.Equal(t, 3, attempts, "should attempt exactly maxAttempts times")
}

func TestRetryWithBackoff_SingleAttempt(t *testing.T) {
	// maxAttempts=1 means one try, zero retries.
	attempts := 0
	err := RetryWithBackoff(1, func(error) bool { return true }, func() error {
		attempts++
		return errors.New("fail")
	})
	assert.Error(t, err)
	assert.Equal(t, 1, attempts)
}
