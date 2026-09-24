//go:build windows

package env

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessNameFindsCurrentProcessAndRejectsMissingPID(t *testing.T) {
	name, err := processName(os.Getpid())
	require.NoError(t, err)
	assert.NotEmpty(t, name)

	_, err = processName(-1)
	require.Error(t, err)
}
