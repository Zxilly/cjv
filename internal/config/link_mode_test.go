package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLinkModeDefaultsAndValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.toml")
	require.NoError(t, os.WriteFile(path, []byte("version = 1\n"), 0o644))
	settings, err := LoadSettings(path)
	require.NoError(t, err)
	assert.Equal(t, "hardlink", settings.LinkMode)
	for _, mode := range []string{"copy", "hardlink"} {
		require.NoError(t, os.WriteFile(path, []byte("version = 1\nlink_mode = \""+mode+"\"\n"), 0o644))
		settings, err := LoadSettings(path)
		require.NoError(t, err)
		assert.Equal(t, mode, settings.LinkMode)
	}
	require.NoError(t, os.WriteFile(path, []byte("version = 1\nlink_mode = \"symlink\"\n"), 0o644))
	_, err = LoadSettings(path)
	require.ErrorContains(t, err, "invalid link_mode")
}
