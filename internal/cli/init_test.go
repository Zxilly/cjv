package cli

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/sdktools"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunInitNonInteractiveNoToolchainWritesManagedFiles(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)
	config.ResetDefaultSettingsFileCache()

	app.initYes = true
	app.initDefaultToolchain = "none"
	app.initNoModifyPath = true
	t.Cleanup(func() {
		config.ResetDefaultSettingsFileCache()
	})

	err := app.runInit(&cobra.Command{}, nil)

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(home, "bin", sdktools.CjvBinaryName()))
	if runtime.GOOS == "windows" {
		assert.FileExists(t, filepath.Join(home, "env.ps1"))
		assert.FileExists(t, filepath.Join(home, "env.bat"))
	} else {
		assert.FileExists(t, filepath.Join(home, "env"))
	}
	for _, tool := range sdktools.AllProxyTools() {
		assert.FileExists(t, filepath.Join(home, "bin", sdktools.PlatformBinaryName(tool)))
	}

	settings, err := config.LoadSettings(filepath.Join(home, ".cjv", "settings.toml"))
	require.NoError(t, err)
	assert.Equal(t, config.DefaultManifestURL, settings.ManifestURL)
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "lts-1.0.5"))
}

func TestInstallInitRestoresHomeEnvironment(t *testing.T) {
	for _, state := range []string{"unset", "empty", "custom"} {
		for _, fail := range []bool{false, true} {
			name := state + "/success"
			if fail {
				name = state + "/bin-file-failure"
			}
			t.Run(name, func(t *testing.T) {
				userHome := t.TempDir()
				config.IsolateForTest(t, userHome)
				t.Setenv(config.EnvNoPathSetup, "1")
				t.Setenv(config.EnvHome, "")
				switch state {
				case "unset":
					require.NoError(t, os.Unsetenv(config.EnvHome))
				case "custom":
					t.Setenv(config.EnvHome, filepath.Join(userHome, "original"))
				}
				before, wasPresent := os.LookupEnv(config.EnvHome)
				initialHome, err := config.Home()
				require.NoError(t, err)
				selectedHome := filepath.Join(userHome, "selected")
				if fail {
					require.NoError(t, os.MkdirAll(selectedHome, 0o755))
					require.NoError(t, os.WriteFile(filepath.Join(selectedHome, "bin"), []byte("keep this file"), 0o644))
				}

				app := newApplication("dev", "")
				_, err = captureStdout(t, func() error {
					return app.installInit(t.Context(), initialHome, initCustomizeOptions{
						home:      selectedHome,
						toolchain: "none",
					})
				})
				if fail {
					require.Error(t, err)
					contents, readErr := os.ReadFile(filepath.Join(selectedHome, "bin"))
					require.NoError(t, readErr)
					assert.Equal(t, "keep this file", string(contents))
				} else {
					require.NoError(t, err)
					assert.FileExists(t, filepath.Join(selectedHome, "bin", sdktools.CjvBinaryName()))
					assert.FileExists(t, filepath.Join(selectedHome, "bin", sdktools.PlatformBinaryName("cjc")))
				}
				after, isPresent := os.LookupEnv(config.EnvHome)
				assert.Equal(t, wasPresent, isPresent, "preserve whether CJV_HOME existed")
				assert.Equal(t, before, after, "preserve the caller's CJV_HOME")
				path, err := config.SettingsPath()
				require.NoError(t, err)
				settings, err := config.LoadSettings(path)
				require.NoError(t, err)
				assert.Equal(t, selectedHome, settings.Home, "selected home remains persisted")

				// A later invocation must be able to change the persisted home
				// without an environment override left behind by init masking it.
				if state != "custom" {
					nextHome := filepath.Join(userHome, "next")
					next := newApplication("dev", "")
					next.rootCmd.SetOut(io.Discard)
					require.NoError(t, next.execute([]string{"set", "home", nextHome}))
					resolved, source, resolveErr := config.ResolveHomeWithSource()
					require.NoError(t, resolveErr)
					assert.Equal(t, nextHome, resolved)
					assert.Equal(t, config.HomeSourcePersisted, source)
				}
			})
		}
	}
}

func TestRunInitContinuesWhenDefaultToolchainInstallFails(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)
	config.ResetDefaultSettingsFileCache()

	app.initYes = true
	app.initDefaultToolchain = "local-sdk"
	app.initNoModifyPath = true
	originalNoPathSetup := os.Getenv(config.EnvNoPathSetup)
	t.Cleanup(func() {
		config.ResetDefaultSettingsFileCache()
	})

	err := app.runInit(&cobra.Command{}, nil)

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(home, "bin", sdktools.CjvBinaryName()))
	assert.Equal(t, originalNoPathSetup, os.Getenv(config.EnvNoPathSetup))
}

func TestRunInitCoversAlreadyInstalledAndModifyPathBranches(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)
	config.ResetDefaultSettingsFileCache()

	var pathConfigured bool
	app.initYes = true
	app.initDefaultToolchain = "none"
	app.initNoModifyPath = false
	app.ensurePathConfiguredFn = func() { pathConfigured = true }
	t.Cleanup(func() {
		config.ResetDefaultSettingsFileCache()
	})

	require.NoError(t, app.runInit(&cobra.Command{}, nil))
	require.True(t, pathConfigured)

	pathConfigured = false
	require.NoError(t, app.runInit(&cobra.Command{}, nil))
	require.True(t, pathConfigured)

	assert.NotEmpty(t, yesNoStr(true))
	assert.NotEmpty(t, yesNoStr(false))
}

func TestRunInitPassesConfiguredComponentsToDefaultToolchainInstall(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)
	config.ResetDefaultSettingsFileCache()

	app.initYes = true
	app.initDefaultToolchain = "sts"
	app.initNoModifyPath = true
	app.initComponents = []string{"stdx", "docs"}

	var gotInput string
	var gotComponents []string
	app.installToolchainWithExtrasFn = func(ctx context.Context, input string, targets, components []string, force bool) error {
		gotInput = input
		gotComponents = append([]string(nil), components...)
		return nil
	}

	t.Cleanup(func() {
		config.ResetDefaultSettingsFileCache()
	})

	err := app.runInit(&cobra.Command{}, nil)

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
