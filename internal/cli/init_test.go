package cli

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/Zxilly/cjv/internal/testutil"
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

// observeInitPathSetup points HOME at a temporary shell config so the test
// can see the PATH block init writes. Windows configures PATH in the registry
// instead, which reachable's guarded tests and the integration tests cover.
func observeInitPathSetup(t *testing.T, cjvHome string) func() {
	t.Helper()
	if runtime.GOOS == "windows" {
		return func() {}
	}
	userHome := t.TempDir()
	rc := filepath.Join(userHome, ".bashrc")
	require.NoError(t, os.WriteFile(rc, []byte("# existing\n"), 0o644))
	t.Setenv("HOME", userHome)
	t.Setenv(config.EnvNoPathSetup, "")
	return func() {
		t.Helper()
		data, err := os.ReadFile(rc)
		require.NoError(t, err)
		assert.Equal(t, 1, strings.Count(string(data), "# cjv (managed by cjv, do not edit)"))
		assert.Contains(t, string(data), filepath.Join(cjvHome, "bin"))
	}
}

func TestRunInitCoversAlreadyInstalledAndModifyPathBranches(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)
	config.ResetDefaultSettingsFileCache()

	app.initYes = true
	app.initDefaultToolchain = "none"
	app.initNoModifyPath = false
	assertPathConfigured := observeInitPathSetup(t, home)
	t.Cleanup(func() {
		config.ResetDefaultSettingsFileCache()
	})

	require.NoError(t, app.runInit(&cobra.Command{}, nil))
	assertPathConfigured()

	// The second run takes the already-installed branch and leaves the
	// single PATH block in place.
	require.NoError(t, app.runInit(&cobra.Command{}, nil))
	assertPathConfigured()

	assert.NotEmpty(t, yesNoStr(true))
	assert.NotEmpty(t, yesNoStr(false))
}

func TestRunInitPassesConfiguredComponentsToDefaultToolchainInstall(t *testing.T) {
	app := newApplication("dev", "")
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv(config.EnvDistServer, "")
	t.Setenv(config.EnvNoPathSetup, "1")
	server := testutil.SplitNightlyMockServer(t)
	settings := config.DefaultSettings()
	settings.DistServer = server.URL + "/corp/cjv"
	settingsPath, err := config.SettingsPath()
	require.NoError(t, err)
	require.NoError(t, config.SaveSettings(&settings, settingsPath))
	config.ResetDefaultSettingsFileCache()

	app.initYes = true
	app.initDefaultToolchain = "nightly"
	app.initNoModifyPath = true
	app.initComponents = []string{"docs"}

	t.Cleanup(func() {
		config.ResetDefaultSettingsFileCache()
	})

	require.NoError(t, app.runInit(&cobra.Command{}, nil))

	const name = "nightly-1.2.0-alpha.20260822010101"
	assert.DirExists(t, filepath.Join(home, "toolchains", name))
	assert.FileExists(t, filepath.Join(home, "docs", name, "main", "index.html"),
		"the configured component is installed into the default toolchain")
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
