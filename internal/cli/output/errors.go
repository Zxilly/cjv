package output

import "github.com/Zxilly/cjv/internal/cjverr"

type errorPayload = cjverr.ErrorInfo

// buildErrorPayload maps a Go error into the JSON error envelope payload.
// The same mapping describes nested automatic-install causes.
func buildErrorPayload(err error) errorPayload {
	return cjverr.Describe(err)
}
