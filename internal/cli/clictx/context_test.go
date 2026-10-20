package clictx

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveToolchainArgValidationAndActiveFallback(t *testing.T) {
	_, _, err := ResolveToolchainArg("+bad")
	require.Error(t, err)

	home := t.TempDir()
	configtest.Isolate(t, home)
	require.NoError(t, config.EnsureDirs())

	_, _, err = ResolveToolchainArg("lts-1.0.5")
	require.Error(t, err)

	tcName := "lts-1.0.5"
	tcDir := filepath.Join(home, "toolchains", tcName)
	require.NoError(t, os.MkdirAll(tcDir, 0o755))
	settings := config.DefaultSettings()
	settings.DefaultToolchain = tcName
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))
	config.ResetDefaultSettingsFileCache()

	gotDir, gotName, err := ResolveToolchainArg("")

	require.NoError(t, err)
	assert.Equal(t, tcDir, gotDir)
	assert.Equal(t, tcName, gotName.String())
}

func TestToolchainPrefersFlagOverSelector(t *testing.T) {
	c := New("dev", "")
	assert.Empty(t, c.Toolchain(""))
	c.Selector = "sts"
	assert.Equal(t, "sts", c.Toolchain(""))
	assert.Equal(t, "lts", c.Toolchain("lts"))
}
