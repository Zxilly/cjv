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

func TestLinkModeInheritsFallbackWithoutPersistingIt(t *testing.T) {
	for _, explicit := range []bool{false, true} {
		t.Run(map[bool]string{false: "inherited", true: "explicit"}[explicit], func(t *testing.T) {
			base := t.TempDir()
			user, fallback := filepath.Join(base, "user.toml"), filepath.Join(base, "fallback.toml")
			t.Setenv(EnvFallbackSettings, fallback)
			require.NoError(t, os.WriteFile(fallback, []byte("link_mode = \"copy\"\n"), 0o644))
			data := "version = 1\n"
			if explicit {
				data += "link_mode = \"hardlink\"\n"
			}
			require.NoError(t, os.WriteFile(user, []byte(data), 0o644))
			sf := NewSettingsFile(user)
			settings, err := sf.Load()
			require.NoError(t, err)
			assert.Equal(t, map[bool]string{false: "copy", true: "hardlink"}[explicit], settings.LinkMode)
			choice := "sts"
			_, err = sf.Update(SettingsUpdate{DefaultToolchain: &choice})
			require.NoError(t, err)
			settings, err = sf.Load()
			require.NoError(t, err)
			require.NoError(t, sf.Save(settings))
			persisted, err := os.ReadFile(user)
			require.NoError(t, err)
			if explicit {
				assert.Contains(t, string(persisted), "link_mode = \"hardlink\"")
			} else {
				assert.NotContains(t, string(persisted), "link_mode")
			}
		})
	}
}
