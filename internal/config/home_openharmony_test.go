//go:build openharmony

package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenHarmonyRequiresUserHome(t *testing.T) {
	for _, home := range []string{"", ".", "/", "relative"} {
		assert.Error(t, validateUserHome(home), home)
	}
	require.NoError(t, validateUserHome(t.TempDir()))
}
