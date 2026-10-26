package dist

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
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
	var err error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if attempt > 0 {
			slog.Info("retrying download", "attempt", attempt+1, "max", maxAttempts)
		}
		err = transfer()
		if err == nil {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		var permanent *nonRetriableError
		if errors.As(err, &permanent) || attempt == maxAttempts-1 {
			return err
		}
		delay := 250 * time.Millisecond * time.Duration(1<<min(attempt, 5))
		var requested *retryAfterError
		if errors.As(err, &requested) && requested.delay > delay {
			delay = requested.delay
		}
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}
	return err
}

type retryAfterError struct {
	err   error
	delay time.Duration
}

func (e *retryAfterError) Error() string { return e.err.Error() }
func (e *retryAfterError) Unwrap() error { return e.err }
func retryAfter(value string) time.Duration {
	if seconds, err := strconv.Atoi(value); err == nil && seconds > 0 {
		return time.Duration(min(seconds, 60)) * time.Second
	}
	if deadline, err := http.ParseTime(value); err == nil {
		return min(max(time.Until(deadline), 0), time.Minute)
	}
	return 0
}

type transferHTTPStatusError struct {
	status int
	url    string
}

func (e *transferHTTPStatusError) Error() string {
	return fmt.Sprintf("HTTP %d for %s", e.status, redactURL(e.url))
}

func (e *transferHTTPStatusError) Code() cjverr.ErrorCode { return cjverr.ErrorCodeHTTP }
func (e *transferHTTPStatusError) Details() map[string]any {
	return map[string]any{"status_code": e.status, "url": redactURL(e.url)}
}

func transferHTTPError(status int, url string) error {
	err := &transferHTTPStatusError{status: status, url: url}
	if isNonRetriableHTTPStatus(status) {
		return &nonRetriableError{err: err}
	}
	return err
}

func transferResponseError(resp *http.Response, url string) error {
	err := transferHTTPError(resp.StatusCode, url)
	if !isNonRetriableHTTPStatus(resp.StatusCode) {
		return &retryAfterError{err: err, delay: retryAfter(resp.Header.Get("Retry-After"))}
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
