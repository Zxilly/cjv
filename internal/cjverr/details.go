package cjverr

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net"
	"net/url"
)

// ErrorInfo is shared by the CLI envelope and nested automatic-install causes.
type ErrorInfo struct {
	Code    ErrorCode      `json:"code"`
	Message string         `json:"message"`
	Details map[string]any `json:"details"`
}

// Describe preserves a coded error's identity, including when wrapped, and
// assigns stable codes to cancellation and transport failures.
func Describe(err error) ErrorInfo {
	if coded, ok := errors.AsType[Coded](err); ok {
		return ErrorInfo{Code: coded.Code(), Message: err.Error(), Details: coded.Details()}
	}
	code := ErrorCodeUnknown
	switch {
	case errors.Is(err, context.Canceled):
		code = ErrorCodeCanceled
	case errors.Is(err, context.DeadlineExceeded):
		code = ErrorCodeDeadlineExceeded
	default:
		if _, ok := errors.AsType[net.Error](err); ok {
			code = ErrorCodeNetwork
		}
	}
	return ErrorInfo{Code: code, Message: err.Error(), Details: map[string]any{}}
}

func messageWithCause(message string, cause error) string {
	if cause != nil {
		return message + ": " + cause.Error()
	}
	return message
}

func detailsWithCause(details map[string]any, cause error) map[string]any {
	if cause != nil {
		details["cause"] = Describe(cause)
	}
	return details
}

// OfflineCacheMissError distinguishes missing reusable data from an SDK that
// simply has not been installed. It retains the filesystem missing sentinel.
type OfflineCacheMissError struct {
	Kind string
	URL  string
}

func (e *OfflineCacheMissError) Error() string {
	return fmt.Sprintf("offline mode: %s for %s is not cached: %v", e.Kind, redactedURL(e.URL), fs.ErrNotExist)
}
func (e *OfflineCacheMissError) Unwrap() error   { return fs.ErrNotExist }
func (e *OfflineCacheMissError) Code() ErrorCode { return ErrorCodeOfflineCacheMiss }
func (e *OfflineCacheMissError) Details() map[string]any {
	return map[string]any{"kind": e.Kind, "url": redactedURL(e.URL)}
}

type NetworkDisabledError struct{ URL string }

func (e *NetworkDisabledError) Error() string {
	return fmt.Sprintf("offline mode: network access to %s is disabled", redactedURL(e.URL))
}
func (e *NetworkDisabledError) Code() ErrorCode { return ErrorCodeNetworkDisabled }
func (e *NetworkDisabledError) Details() map[string]any {
	return map[string]any{"url": redactedURL(e.URL)}
}

func redactedURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "<invalid URL>"
	}
	return parsed.Redacted()
}
