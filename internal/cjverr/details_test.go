package cjverr

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMissingComponentRetainsWrappedCause(t *testing.T) {
	cause := fmt.Errorf("install docs: %w", context.Canceled)
	err := &ComponentNotInstalledError{Toolchain: "lts", Component: "docs", Cause: cause}
	require.ErrorIs(t, err, context.Canceled)
	info := Describe(fmt.Errorf("resolve environment: %w", err))
	assert.Equal(t, ErrorCodeComponentNotInstalled, info.Code)
	assert.Equal(t, "lts", info.Details["toolchain"])
	assert.Equal(t, "docs", info.Details["component"])
	assert.Equal(t, ErrorCodeCanceled, info.Details["cause"].(ErrorInfo).Code)
	assert.Contains(t, info.Message, cause.Error())
	assert.NotContains(t, (&ComponentNotInstalledError{}).Details(), "cause")
	assert.NotContains(t, (&ToolchainNotInstalledError{}).Details(), "cause")
}

func TestOfflineErrorsRedactCredentialsInMessageAndDetails(t *testing.T) {
	rawURL := "https://user:secret@example.invalid/sdk.zip"
	missing := &OfflineCacheMissError{Kind: "artifact", URL: rawURL}
	require.ErrorIs(t, missing, fs.ErrNotExist)
	for _, err := range []error{missing, &NetworkDisabledError{URL: rawURL}} {
		data, encodeErr := json.Marshal(Describe(err))
		require.NoError(t, encodeErr)
		assert.NotContains(t, string(data), "secret")
		assert.Contains(t, string(data), "example.invalid")
	}
}
