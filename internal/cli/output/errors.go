package output

import (
	"errors"

	"github.com/Zxilly/cjv/internal/cjverr"
)

type errorPayload struct {
	Code    cjverr.ErrorCode `json:"code"`
	Message string           `json:"message"`
	Details map[string]any   `json:"details"`
}

// buildErrorPayload maps a Go error into the JSON error envelope payload.
// Errors that implement cjverr.Coded supply their own code and details;
// everything else falls back to UNKNOWN with empty details.
func buildErrorPayload(err error) errorPayload {
	if coded, ok := errors.AsType[cjverr.Coded](err); ok {
		return errorPayload{
			Code:    coded.Code(),
			Message: err.Error(),
			Details: coded.Details(),
		}
	}
	return errorPayload{
		Code:    cjverr.ErrorCodeUnknown,
		Message: err.Error(),
		Details: map[string]any{},
	}
}
