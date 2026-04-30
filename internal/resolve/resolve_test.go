package resolve

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShouldAutoInstall_RespectsExplicitSetting(t *testing.T) {
	s := config.DefaultSettings()

	s.AutoInstall = true
	assert.True(t, shouldAutoInstall(&s), "should auto-install when explicitly enabled")

	s.AutoInstall = false
	assert.False(t, shouldAutoInstall(&s), "should not auto-install when explicitly disabled")
}

func TestShouldAutoInstall_NilSettingsReturnsFalse(t *testing.T) {
	assert.False(t, shouldAutoInstall(nil), "should return false when settings is nil")
}

func TestActiveRejectsTargetVariantAsActiveToolchain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	t.Setenv("CJV_TOOLCHAIN", "")
	require.NoError(t, config.EnsureDirs())

	key, err := dist.CurrentPlatformKeyWithTarget("", "ohos")
	require.NoError(t, err)
	name := toolchain.ToolchainName{
		Channel:     toolchain.STS,
		Version:     "2.0.0",
		PlatformKey: key,
	}.String()
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", name), 0o755))
	require.NoError(t, config.SaveSettings(&config.Settings{
		Version:          1,
		DefaultToolchain: name,
		AutoInstall:      true,
		Overrides:        map[string]string{},
	}, home+"/settings.toml"))
	t.Setenv("CJV_TOOLCHAIN", name)

	_, err = Active(t.Context(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "target variant")
}
