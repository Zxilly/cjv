package resolve

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/component"
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

func TestResolveNamePrefersOverrideThenEnvironment(t *testing.T) {
	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"

	name, source, targets, components, err := resolveName(&settings, nil, "sts-2.0.0")
	require.NoError(t, err)
	assert.Equal(t, "sts-2.0.0", name)
	assert.Equal(t, config.SourceUnknown, source)
	assert.Nil(t, targets)
	assert.Nil(t, components)

	t.Setenv(config.EnvToolchain, "nightly-202501010000")
	name, source, targets, components, err = resolveName(&settings, nil, "")
	require.NoError(t, err)
	assert.Equal(t, "nightly-202501010000", name)
	assert.Equal(t, config.SourceEnv, source)
	assert.Nil(t, targets)
	assert.Nil(t, components)
}

func TestResolveNameReturnsSettingsErrorWhenNoOverride(t *testing.T) {
	expected := errors.New("settings failed")

	_, _, _, _, err := resolveName(nil, expected, "")

	assert.ErrorIs(t, err, expected)
}

func TestActiveRejectsTargetVariantAsActiveToolchain(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	t.Setenv("CJV_TOOLCHAIN", "")
	require.NoError(t, config.EnsureDirs())

	key, err := dist.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	name := toolchain.ToolchainName{
		Channel:     toolchain.STS,
		Version:     "2.0.0",
		Target: key,
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
		key, err := dist.CurrentTargetTuple(settings.DefaultHost, "ohos")
		require.NoError(t, err)
		targetName := toolchain.ToolchainName{Channel: toolchain.STS, Version: "2.0.0", Target: key}.String()
		return os.MkdirAll(filepath.Join(home, "toolchains", targetName), 0o755)
	}
	AutoInstallComponentsFunc = func(ctx context.Context, input string, components []string) error {
		gotComponentInput = input
		gotComponents = append([]string(nil), components...)
		return component.WriteManifest(hostDir, component.Docs, []string{"index.html"})
	}
	t.Cleanup(func() {
		AutoInstallFunc = oldInstall
		AutoInstallComponentsFunc = oldComponents
	})

	active, err := Active(context.Background(), "")

	require.NoError(t, err)
	assert.Equal(t, hostName, active.Name)
	assert.Equal(t, config.SourceToolchainFile, active.Source)
	assert.Equal(t, []string{"ohos"}, active.Targets)
	assert.Equal(t, []string{"docs"}, active.Components)
	assert.Equal(t, hostName, gotInput)
	assert.Equal(t, []string{"ohos"}, gotTargets)
	assert.Equal(t, hostName, gotComponentInput)
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

	err := ensureComponents(context.Background(), tcName, filepath.Join(home, "toolchains", tcName), &settings, []string{"docs"})

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
	assert.Equal(t, "lts-1.0.5", active.Name)
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

	err := ensureTargets(context.Background(), tcName, tcDir, &settings, []string{"ohos"})

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

	err := ensureTargets(context.Background(), tcName, tcDir, &settings, []string{"ohos"})
	require.Error(t, err)

	AutoInstallFunc = func(ctx context.Context, input string, targets []string) error {
		return nil
	}
	err = ensureTargets(context.Background(), tcName, tcDir, &settings, []string{"ohos"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ohos")
}

func TestEnsureComponentsInvalidAndAutoInstallFailures(t *testing.T) {
	tcDir := t.TempDir()
	settings := config.DefaultSettings()

	require.Error(t, ensureComponents(context.Background(), "lts-1.0.5", tcDir, &settings, []string{"unknown"}))

	settings.AutoInstall = true
	oldComponents := AutoInstallComponentsFunc
	t.Cleanup(func() { AutoInstallComponentsFunc = oldComponents })
	AutoInstallComponentsFunc = func(ctx context.Context, input string, components []string) error {
		return os.ErrPermission
	}

	err := ensureComponents(context.Background(), "lts-1.0.5", tcDir, &settings, []string{"docs"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docs")

	AutoInstallComponentsFunc = func(ctx context.Context, input string, components []string) error {
		return nil
	}
	err = ensureComponents(context.Background(), "lts-1.0.5", tcDir, &settings, []string{"docs"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "docs")
}
