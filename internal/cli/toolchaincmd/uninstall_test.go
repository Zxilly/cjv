package toolchaincmd

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Zxilly/cjv/internal/cli/clitest"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/fstx"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for the uninstall command. Settings cleanup after removal is
// lifecycle behaviour, tested in internal/lifecycle.

// emptyHome isolates a recovered home with an empty toolchains directory.
func emptyHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	configtest.Isolate(t, home)
	require.NoError(t, toolchain.RecoverHome())
	require.NoError(t, config.EnsureDirs())
	return home
}

func TestRunUninstallRemovesOnlyNamedToolchain(t *testing.T) {
	home := validManifestHome(t)
	ctx := newTestContext()
	clitest.InstallToolchain(t, ctx, "lts")
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "sts-2.0.0"), 0o755))

	require.NoError(t, newUninstall(ctx).run(nil, []string{"sts-2.0.0"}))
	remaining, err := toolchain.ListInstalled()
	require.NoError(t, err)
	assert.Equal(t, []string{"lts"}, remaining)

	require.NoError(t, newUninstall(ctx).run(nil, []string{"lts"}))
	remaining, err = toolchain.ListInstalled()
	require.NoError(t, err)
	assert.Empty(t, remaining)
}

func TestRunUninstallNotInstalled(t *testing.T) {
	emptyHome(t)
	err := newUninstall(newTestContext()).run(nil, []string{"nonexistent-99.99"})
	assert.Error(t, err, "uninstalling a toolchain that is not installed should error")
}

func TestRunUninstallRecoversBeforeCheckingInstallation(t *testing.T) {
	home := emptyHome(t)
	name := "lts-1.0.5"
	dir := filepath.Join(home, "toolchains", name)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "compiler"), []byte("old compiler"), 0o644))
	tx, err := fstx.NewToolchainTransaction(home, name)
	require.NoError(t, err)
	require.NoError(t, tx.RemoveDir(dir))
	assert.NoDirExists(t, dir)
	// The CLI must recover before its confirmation/existence check, not report
	// "not installed" for content that is still owned by a retained journal.
	require.NoError(t, newUninstall(newTestContext()).run(nil, []string{name}))
	assert.NoDirExists(t, dir)
	entries, err := os.ReadDir(filepath.Dir(dir))
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestRunUninstallPreservesSettingsWhenRemoveFails(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows keeps the process working directory locked")
	}
	home := emptyHome(t)

	name := "lts-1.0.5"
	toolchainDir := filepath.Join(home, "toolchains", name)
	require.NoError(t, os.MkdirAll(toolchainDir, 0o755))

	projectDir := filepath.Join(home, "project")
	settings := config.DefaultSettings()
	settings.DefaultToolchain = name
	settings.Overrides[projectDir] = name
	settingsPath := filepath.Join(home, ".cjv", "settings.toml")
	require.NoError(t, config.SaveSettings(&settings, settingsPath))

	wd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(toolchainDir))
	t.Cleanup(func() {
		_ = os.Chdir(wd)
		_ = os.RemoveAll(filepath.Join(home, "toolchains"))
	})

	require.Error(t, newUninstall(newTestContext()).run(nil, []string{name}))
	loaded, err := config.LoadSettings(settingsPath)
	require.NoError(t, err)
	assert.Equal(t, name, loaded.DefaultToolchain)
	assert.Equal(t, name, loaded.Overrides[projectDir])
}
