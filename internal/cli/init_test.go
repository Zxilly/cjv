package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/proxy"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunInitNonInteractiveNoToolchainWritesManagedFiles(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	config.ResetDefaultSettingsFileCache()

	oldYes := initYes
	oldToolchain := initDefaultToolchain
	oldNoModifyPath := initNoModifyPath
	initYes = true
	initDefaultToolchain = "none"
	initNoModifyPath = true
	t.Cleanup(func() {
		initYes = oldYes
		initDefaultToolchain = oldToolchain
		initNoModifyPath = oldNoModifyPath
		config.ResetDefaultSettingsFileCache()
	})

	err := runInit(&cobra.Command{}, nil)

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(home, "bin", proxy.CjvBinaryName()))
	if runtime.GOOS == "windows" {
		assert.FileExists(t, filepath.Join(home, "env.ps1"))
		assert.FileExists(t, filepath.Join(home, "env.bat"))
	} else {
		assert.FileExists(t, filepath.Join(home, "env"))
	}
	for _, tool := range proxy.AllProxyTools() {
		assert.FileExists(t, filepath.Join(home, "bin", proxy.PlatformBinaryName(tool)))
	}

	settings, err := config.LoadSettings(filepath.Join(home, ".cjv", "settings.toml"))
	require.NoError(t, err)
	assert.Equal(t, config.DefaultManifestURL, settings.ManifestURL)
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "lts-1.0.5"))
}

func TestRunInitContinuesWhenDefaultToolchainInstallFails(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	config.ResetDefaultSettingsFileCache()

	oldYes := initYes
	oldToolchain := initDefaultToolchain
	oldNoModifyPath := initNoModifyPath
	initYes = true
	initDefaultToolchain = "local-sdk"
	initNoModifyPath = true
	t.Cleanup(func() {
		initYes = oldYes
		initDefaultToolchain = oldToolchain
		initNoModifyPath = oldNoModifyPath
		_ = os.Unsetenv(config.EnvNoPathSetup)
		config.ResetDefaultSettingsFileCache()
	})

	err := runInit(&cobra.Command{}, nil)

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(home, "bin", proxy.CjvBinaryName()))
	assert.Empty(t, os.Getenv(config.EnvNoPathSetup))
}

func TestRunInitCoversAlreadyInstalledAndModifyPathBranches(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	config.ResetDefaultSettingsFileCache()

	oldYes := initYes
	oldToolchain := initDefaultToolchain
	oldNoModifyPath := initNoModifyPath
	oldEnsurePath := ensurePathConfiguredFn
	var pathConfigured bool
	initYes = true
	initDefaultToolchain = "none"
	initNoModifyPath = false
	ensurePathConfiguredFn = func() { pathConfigured = true }
	t.Cleanup(func() {
		initYes = oldYes
		initDefaultToolchain = oldToolchain
		initNoModifyPath = oldNoModifyPath
		ensurePathConfiguredFn = oldEnsurePath
		config.ResetDefaultSettingsFileCache()
	})

	require.NoError(t, runInit(&cobra.Command{}, nil))
	require.True(t, pathConfigured)

	pathConfigured = false
	require.NoError(t, runInit(&cobra.Command{}, nil))
	require.True(t, pathConfigured)

	assert.NotEmpty(t, yesNoStr(true))
	assert.NotEmpty(t, yesNoStr(false))
}

func TestRunInitPassesConfiguredComponentsToDefaultToolchainInstall(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	config.ResetDefaultSettingsFileCache()

	oldYes := initYes
	oldToolchain := initDefaultToolchain
	oldNoModifyPath := initNoModifyPath
	oldComponents := initComponents
	oldInstall := installToolchainWithExtrasFn
	initYes = true
	initDefaultToolchain = "sts"
	initNoModifyPath = true
	initComponents = []string{"stdx", "docs"}

	var gotInput string
	var gotComponents []string
	installToolchainWithExtrasFn = func(ctx context.Context, input string, targets, components []string, force bool) error {
		gotInput = input
		gotComponents = append([]string(nil), components...)
		return nil
	}

	t.Cleanup(func() {
		initYes = oldYes
		initDefaultToolchain = oldToolchain
		initNoModifyPath = oldNoModifyPath
		initComponents = oldComponents
		installToolchainWithExtrasFn = oldInstall
		config.ResetDefaultSettingsFileCache()
	})

	err := runInit(&cobra.Command{}, nil)

	require.NoError(t, err)
	assert.Equal(t, "sts", gotInput)
	assert.Equal(t, []string{"stdx", "docs"}, gotComponents)
}

func TestRenderInitMarkdown(t *testing.T) {
	rendered, err := renderInitMarkdown("Use `cjv install lts` to install a toolchain.")

	require.NoError(t, err)
	assert.Contains(t, rendered, "cjv install lts")
}

func TestInitCustomizeFormEndToEnd(t *testing.T) {
	userHome := t.TempDir()
	target := filepath.Join(userHome, "custom", "cjv")
	opts := initCustomizeOptions{
		toolchain:  "lts",
		components: []string{"stdx"},
		modifyPath: true,
	}

	form := newInitCustomizeForm(&opts)
	form.SubmitCmd = tea.Quit
	tm := teatest.NewTestModel(t, form, teatest.WithInitialTermSize(100, 40))
	t.Cleanup(func() {
		_ = tm.Quit()
	})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte(i18n.T("InitInstallPathQuestion", nil)))
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))
	tm.Type(target)
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte(i18n.T("InitToolchainQuestion", nil)))
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))
	for range 3 {
		tm.Send(tea.KeyMsg{Type: tea.KeyDown})
	}
	tm.Send(tea.KeyMsg{Type: tea.KeyEnter})

	teatest.WaitFor(t, tm.Output(), func(out []byte) bool {
		return bytes.Contains(out, []byte(i18n.T("InitModifyPathQuestion", nil)))
	}, teatest.WithDuration(2*time.Second), teatest.WithCheckInterval(10*time.Millisecond))
	tm.Type("n")
	tm.FinalModel(t, teatest.WithFinalTimeout(2*time.Second))

	expectedAbs, err := filepath.Abs(target)
	require.NoError(t, err)
	assert.Equal(t, expectedAbs, opts.home)
	assert.Equal(t, "none", opts.toolchain)
	assert.Empty(t, opts.components)
	assert.False(t, opts.modifyPath)
	assert.NoDirExists(t, target)
}

func TestInitHomePathValidationAndActivation(t *testing.T) {
	userHome := t.TempDir()
	config.IsolateForTest(t, userHome)
	config.ResetDefaultSettingsFileCache()
	t.Cleanup(config.ResetDefaultSettingsFileCache)

	filePath := filepath.Join(userHome, "not-a-dir")
	require.NoError(t, os.WriteFile(filePath, []byte("x"), 0o644))
	_, err := normalizeInitHomePath(filePath)
	require.Error(t, err)

	target := filepath.Join(userHome, "custom", "cjv")
	normalized, err := normalizeInitHomePath("  " + target + "  ")
	require.NoError(t, err)
	expectedAbs, err := filepath.Abs(target)
	require.NoError(t, err)
	assert.Equal(t, expectedAbs, normalized)
	assert.NoDirExists(t, target)

	require.NoError(t, activateInitHomePath(normalized))
	assert.DirExists(t, target)
	assert.Equal(t, normalized, os.Getenv(config.EnvHome))

	settingsPath, err := config.SettingsPath()
	require.NoError(t, err)
	settings, err := config.LoadSettings(settingsPath)
	require.NoError(t, err)
	assert.Equal(t, normalized, settings.Home)

	t.Setenv(config.EnvHome, "")
	config.ResetDefaultSettingsFileCache()
	got, src, err := config.ResolveHomeWithSource()
	require.NoError(t, err)
	assert.Equal(t, normalized, got)
	assert.Equal(t, config.HomeSourcePersisted, src)
}
