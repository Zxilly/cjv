// Package retry repeats an operation with Fibonacci backoff until it succeeds
// or its error is no longer worth retrying. fsops uses it for transient
// Windows filesystem errors, dist for network fetches.
package retry

import (
	"context"
	"math/rand/v2"
	"time"
)

// Do calls fn up to maxAttempts times, sleeping with Fibonacci backoff plus
// light jitter between attempts. It stops early on the first error for which
// shouldRetry returns false and returns the last error seen.
func Do(maxAttempts int, shouldRetry func(error) bool, fn func() error) error {
	return DoContext(context.Background(), maxAttempts, shouldRetry, fn)
}

// DoContext also stops before an attempt or during backoff when ctx is canceled.
// The operation remains responsible for respecting ctx while it is running.
func DoContext(ctx context.Context, maxAttempts int, shouldRetry func(error) bool, fn func() error) error {
	var err error
	for i := range maxAttempts {
		if err := ctx.Err(); err != nil {
			return err
		}
		err = fn()
		if err == nil {
			return nil
		}
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if !shouldRetry(err) {
			return err
		}
		if i < maxAttempts-1 {
			timer := time.NewTimer(delay(i))
			select {
			case <-ctx.Done():
				timer.Stop()
				return ctx.Err()
			case <-timer.C:
			}
		}
	}
	return err
}

func delay(attempt int) time.Duration {
	a, b := 10*time.Millisecond, 10*time.Millisecond
	for range attempt {
		a, b = b, a+b
		if b > time.Second {
			b = time.Second
		}
	}

	jitter := time.Duration(rand.Int64N(int64(a/2) + 1))
	return a + jitter
}
