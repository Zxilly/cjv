package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenHarmonyRequiresUserHome(t *testing.T) {
	for _, home := range []string{"", ".", "/", "relative"} {
		assert.Error(t, validateUserHome(home, "openharmony"), home)
	}
	require.NoError(t, validateUserHome(t.TempDir(), "openharmony"))
}

func TestOtherHostsKeepUserHomeSemantics(t *testing.T) {
	for _, goos := range []string{"linux", "darwin", "windows"} {
		require.NoError(t, validateUserHome("/", goos))
	}
}
