package resolve

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/progress"
	"github.com/Zxilly/cjv/internal/sdktools"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/testutil"
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

func TestActivePreparesHostThroughProductionInstaller(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv(config.EnvNoPathSetup, "1")
	server := testutil.ValidMockServer(t)
	sf, err := config.DefaultSettingsFile()
	require.NoError(t, err)
	url := server.URL + "/sdk-versions.json"
	autoInstall := true
	_, err = sf.Update(config.SettingsUpdate{ManifestURL: &url, AutoInstall: &autoInstall})
	require.NoError(t, err)
	require.Nil(t, AutoInstallFunc, "exercise the production lifecycle adapter")

	active, err := Active(t.Context(), "lts")
	require.NoError(t, err)
	assert.Equal(t, "lts", active.Name)
	assert.Equal(t, filepath.Join(home, "toolchains", "lts"), active.Dir)
	installed, err := toolchain.InstalledRelease(active.Dir)
	require.NoError(t, err)
	assert.Equal(t, "lts-1.0.5", installed.String())
}

func TestLegacyMigrationKeepsTargetsAlignedWithSelectedHost(t *testing.T) {
	for _, matching := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing", true: "installed"}[matching], func(t *testing.T) {
			home := t.TempDir()
			config.IsolateForTest(t, home)
			require.NoError(t, config.EnsureDirs())
			tuple, err := sdktarget.CurrentTargetTuple("", "ohos")
			require.NoError(t, err)
			names := []string{"sts-2.0.0", "sts-3.0.0-" + tuple}
			if matching {
				names = append(names, "sts-2.0.0-"+tuple)
			}
			for _, name := range names {
				require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", name), 0o755))
			}
			active, err := ActiveTarget(t.Context(), "sts", "ohos")
			if matching {
				require.NoError(t, err)
				record, err := toolchain.ReadInstallation(active.Dir)
				require.NoError(t, err)
				assert.Equal(t, "sts-2.0.0-"+tuple, record.Release)
			} else {
				require.Error(t, err)
				assert.NoDirExists(t, filepath.Join(home, "toolchains", "sts-"+tuple))
			}
			for _, name := range names {
				assert.DirExists(t, filepath.Join(home, "toolchains", name), "all fixed versions must remain intact")
			}
		})
	}
}

func TestShouldAutoInstall_NilSettingsReturnsFalse(t *testing.T) {
	assert.False(t, shouldAutoInstall(nil), "should return false when settings is nil")
}

func TestActiveRejectsTargetVariantAsActiveToolchain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	t.Setenv("CJV_TOOLCHAIN", "")
	require.NoError(t, config.EnsureDirs())

	key, err := sdktarget.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	name := toolchain.ToolchainName{
		Channel: toolchain.STS,
		Version: "2.0.0",
		Target:  key,
	}.String()
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", name), 0o755))
	parsedName, err := toolchain.ParseToolchainName(name)
	require.NoError(t, err)
	if parsedName.Version == "" && parsedName.Target != "" {
		parsedName.Version = "2.0.0"
		require.NoError(t, toolchain.WriteInstallation(filepath.Join(home, "toolchains", name), toolchain.Installation{Release: parsedName.String(), Tuple: parsedName.Target}))
	}
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

func TestActiveTargetResolvesInstalledTargetSDK(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	t.Setenv("CJV_TOOLCHAIN", "")
	require.NoError(t, config.EnsureDirs())

	hostName := "sts-2.0.0"
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", hostName), 0o755))

	tuple, err := sdktarget.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	targetName := toolchain.ToolchainName{Channel: toolchain.STS, Version: "2.0.0", Target: tuple}.String()
	targetDir := filepath.Join(home, "toolchains", targetName)
	require.NoError(t, os.MkdirAll(targetDir, 0o755))

	active, err := ActiveTarget(t.Context(), hostName, "ohos")
	require.NoError(t, err)
	// Name is the host toolchain identity; Dir/root is the target SDK.
	assert.Equal(t, hostName, active.Name)
	assert.Equal(t, targetDir, active.Dir)
	assert.Equal(t, []string{"ohos"}, active.Targets)
	assert.Nil(t, active.Components)
}

func TestActiveTargetErrorsWhenTargetSDKMissing(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	t.Setenv("CJV_TOOLCHAIN", "")
	require.NoError(t, config.EnsureDirs())

	hostName := "sts-2.0.0"
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", hostName), 0o755))

	_, err := ActiveTarget(t.Context(), hostName, "ohos")
	require.Error(t, err)
	var notInstalled *cjverr.ToolchainNotInstalledError
	assert.True(t, errors.As(err, &notInstalled))
}

func TestActiveAutoInstallsMissingTargetsAndComponents(t *testing.T) {
	home := t.TempDir()
	cwd := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv(config.EnvToolchain, "")
	require.NoError(t, config.EnsureDirs())
	t.Chdir(cwd)

	hostName := "sts-2.0.0"
	hostDir := filepath.Join(home, "toolchains", hostName)
	require.NoError(t, os.MkdirAll(hostDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(cwd, config.ToolchainFileName), []byte(`[toolchain]
channel = "sts"
targets = ["ohos"]
components = ["docs"]
`), 0o644))

	settings := config.DefaultSettings()
	settings.AutoInstall = true
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	oldInstall := AutoInstallFunc
	oldComponents := AutoInstallComponentsFunc
	var gotInput string
	var gotTargets []string
	var gotComponentInput string
	var gotComponents []string
	AutoInstallFunc = func(ctx context.Context, input string, targets []string) error {
		gotInput = input
		gotTargets = append([]string(nil), targets...)
		key, err := sdktarget.CurrentTargetTuple(settings.DefaultHost, "ohos")
		require.NoError(t, err)
		targetName := toolchain.ToolchainName{Channel: toolchain.STS, Target: key}.String()
		dir := filepath.Join(home, "toolchains", targetName)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
		return toolchain.WriteInstallation(dir, toolchain.Installation{Release: "sts-2.0.0-" + key, Tuple: key})
	}
	AutoInstallComponentsFunc = func(ctx context.Context, input string, components []string) error {
		gotComponentInput = input
		gotComponents = append([]string(nil), components...)
		return component.WriteManifest(filepath.Join(home, "toolchains", "sts"), component.Docs, []string{"index.html"})
	}
	t.Cleanup(func() {
		AutoInstallFunc = oldInstall
		AutoInstallComponentsFunc = oldComponents
	})

	active, err := Active(context.Background(), "")

	require.NoError(t, err)
	assert.Equal(t, "sts", active.Name)
	assert.Equal(t, config.SourceToolchainFile, active.Source)
	assert.Equal(t, []string{"ohos"}, active.Targets)
	assert.Equal(t, []string{"docs"}, active.Components)
	assert.Equal(t, "sts", gotInput)
	assert.Equal(t, []string{"ohos"}, gotTargets)
	assert.Equal(t, "sts", gotComponentInput)
	assert.Equal(t, []string{"docs"}, gotComponents)
}

func TestActiveReportsMissingComponentWhenAutoInstallDisabled(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv(config.EnvToolchain, "")
	require.NoError(t, config.EnsureDirs())

	tcName := "lts-1.0.5"
	require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", tcName), 0o755))
	settings := config.DefaultSettings()
	settings.DefaultToolchain = tcName
	settings.AutoInstall = false
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	err := ensureComponents(context.Background(), tcName, filepath.Join(home, "toolchains", tcName), &settings, []string{"docs"}, progress.Discard)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "docs")
}

func TestActiveAutoInstallsMissingHostToolchain(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv(config.EnvToolchain, "")
	require.NoError(t, config.EnsureDirs())

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	settings.AutoInstall = true
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	oldInstall := AutoInstallFunc
	var gotInput string
	AutoInstallFunc = func(ctx context.Context, input string, targets []string) error {
		gotInput = input
		return os.MkdirAll(filepath.Join(home, "toolchains", input), 0o755)
	}
	t.Cleanup(func() { AutoInstallFunc = oldInstall })

	active, err := Active(context.Background(), "")

	require.NoError(t, err)
	assert.Equal(t, "lts-1.0.5", active.Name)
	assert.Equal(t, "lts-1.0.5", gotInput)
}

// Proxy auto-install runs the same installation flow as `cjv install`: the
// managed cjv binary and the proxy links are established, not just the SDK
// directory, so the freshly installed toolchain is reachable through bin/.
func TestActiveAutoInstallCreatesManagedBinaryAndProxyLinks(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv(config.EnvToolchain, "")
	t.Setenv(config.EnvDistServer, "")
	require.NoError(t, config.EnsureDirs())
	server := testutil.MockDistServer(t)

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts"
	settings.AutoInstall = true
	settings.ManifestURL = server.URL + "/sdk-versions.json"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	active, err := Active(context.Background(), "")

	require.NoError(t, err)
	assert.Equal(t, "lts", active.Name)
	assert.FileExists(t, filepath.Join(home, "bin", sdktools.CjvBinaryName()))
	for _, tool := range sdktools.AllProxyTools() {
		assert.FileExists(t, filepath.Join(home, "bin", sdktools.PlatformBinaryName(tool)),
			"proxy link for %q should exist after auto-install", tool)
	}
}

func TestActiveRunsToolchainRecoveryBeforeResolving(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv(config.EnvToolchain, "")
	require.NoError(t, config.EnsureDirs())

	backup := filepath.Join(home, "toolchains", ".fstx-crash", "0-lts-1.0.5")
	require.NoError(t, os.MkdirAll(backup, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(backup, "release.txt"), []byte("old"), 0o644))

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts"
	settings.AutoInstall = false
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	active, err := Active(context.Background(), "")

	require.NoError(t, err)
	assert.Equal(t, "lts", active.Name)
	assert.FileExists(t, filepath.Join(home, "toolchains", "lts-1.0.5", "release.txt"))
}

func TestActiveAutoInstallFailureReportsToolchainMissing(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv(config.EnvToolchain, "")
	require.NoError(t, config.EnsureDirs())

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	settings.AutoInstall = true
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	oldInstall := AutoInstallFunc
	AutoInstallFunc = func(ctx context.Context, input string, targets []string) error {
		return os.ErrPermission
	}
	t.Cleanup(func() { AutoInstallFunc = oldInstall })

	_, err := Active(context.Background(), "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "lts-1.0.5")
}

func TestEnsureTargetsReportsMissingWhenAutoInstallDisabled(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)
	require.NoError(t, config.EnsureDirs())

	tcName := "sts-2.0.0"
	tcDir := filepath.Join(home, "toolchains", tcName)
	require.NoError(t, os.MkdirAll(tcDir, 0o755))
	settings := config.DefaultSettings()
	settings.AutoInstall = false

	err := ensureTargets(context.Background(), tcName, tcDir, &settings, []string{"ohos"}, progress.Discard)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ohos")
}

func TestEnsureTargetsAutoInstallFailureAndMissingResult(t *testing.T) {
	home := t.TempDir()
	t.Setenv(config.EnvHome, home)
	require.NoError(t, config.EnsureDirs())

	tcName := "sts-2.0.0"
	tcDir := filepath.Join(home, "toolchains", tcName)
	require.NoError(t, os.MkdirAll(tcDir, 0o755))
	settings := config.DefaultSettings()
	settings.AutoInstall = true

	oldInstall := AutoInstallFunc
	t.Cleanup(func() { AutoInstallFunc = oldInstall })
	AutoInstallFunc = func(ctx context.Context, input string, targets []string) error {
		return os.ErrPermission
	}

	err := ensureTargets(context.Background(), tcName, tcDir, &settings, []string{"ohos"}, progress.Discard)
	require.Error(t, err)

	AutoInstallFunc = func(ctx context.Context, input string, targets []string) error {
		return nil
	}
	err = ensureTargets(context.Background(), tcName, tcDir, &settings, []string{"ohos"}, progress.Discard)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ohos")
}

func TestEnsureComponentsInvalidAndAutoInstallFailures(t *testing.T) {
	tcDir := t.TempDir()
	settings := config.DefaultSettings()

	require.Error(t, ensureComponents(context.Background(), "lts-1.0.5", tcDir, &settings, []string{"unknown"}, progress.Discard))

	settings.AutoInstall = true
	oldComponents := AutoInstallComponentsFunc
	t.Cleanup(func() { AutoInstallComponentsFunc = oldComponents })
	AutoInstallComponentsFunc = func(ctx context.Context, input string, components []string) error {
		return os.ErrPermission
	}

	err := ensureComponents(context.Background(), "lts-1.0.5", tcDir, &settings, []string{"docs"}, progress.Discard)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docs")

	AutoInstallComponentsFunc = func(ctx context.Context, input string, components []string) error {
		return nil
	}
	err = ensureComponents(context.Background(), "lts-1.0.5", tcDir, &settings, []string{"docs"}, progress.Discard)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docs")
}
