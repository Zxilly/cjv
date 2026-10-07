package dist

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/retry"
)

// nonRetriableError marks failures that another transfer cannot repair.
type nonRetriableError struct {
	err error
}

func (e *nonRetriableError) Error() string { return e.err.Error() }
func (e *nonRetriableError) Unwrap() error { return e.err }

func getMaxDownloadRetries() int {
	if s := os.Getenv(config.EnvMaxRetries); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n >= 0 {
			return n
		}
	}
	return 3
}

// retryTransfer owns the stopping and waiting policy shared by archives and
// checksum metadata. Each operation retains its own partial-file lifecycle.
func retryTransfer(ctx context.Context, transfer func() error) error {
	maxAttempts := getMaxDownloadRetries() + 1
	attempt := 0
	return retry.DoContext(ctx, maxAttempts, func(err error) bool {
		var permanent *nonRetriableError
		return !errors.As(err, &permanent)
	}, func() error {
		attempt++
		if attempt > 1 {
			slog.Info("retrying download", "attempt", attempt, "max", maxAttempts)
		}
		return transfer()
	})
}

func transferHTTPError(status int, url string) error {
	err := fmt.Errorf("HTTP %d for %s", status, url)
	if isNonRetriableHTTPStatus(status) {
		return &nonRetriableError{err: err}
	}
	return err
}

func isNonRetriableHTTPStatus(statusCode int) bool {
	if statusCode < http.StatusBadRequest || statusCode >= http.StatusInternalServerError {
		return false
	}
	switch statusCode {
	case http.StatusRequestTimeout, http.StatusTooManyRequests:
		return false
	default:
		return true
	}
}
