package cli

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	componentlib "github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupComponentCLITest(t *testing.T, tcName string) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	config.ResetDefaultSettingsFileCache()
	tcDir := filepath.Join(home, "toolchains", tcName)
	require.NoError(t, os.MkdirAll(tcDir, 0o755))
	return tcDir
}

func TestRunComponentRemoveBestEffort(t *testing.T) {
	tcName := "lts-1.0.5"
	tcDir := setupComponentCLITest(t, tcName)
	roots, err := componentlib.RootsFor(tcName)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(roots.StdxDir, "dynamic"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(roots.StdxDir, "dynamic", "libfoo.so"), []byte("x"), 0o644))
	require.NoError(t, componentlib.WriteManifest(tcDir, componentlib.Stdx, []string{"dynamic/libfoo.so"}))

	oldToolchain := componentToolchain
	componentToolchain = tcName
	defer func() { componentToolchain = oldToolchain }()

	err = runComponentRemove(&cobra.Command{}, []string{"bogus", "stdx", "docs"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus")
	assert.Contains(t, err.Error(), "docs")
	assert.False(t, componentlib.IsInstalled(tcDir, componentlib.Stdx))
	assert.NoFileExists(t, filepath.Join(roots.StdxDir, "dynamic", "libfoo.so"))
}

func TestResolveToolchainArgValidationAndActiveFallback(t *testing.T) {
	_, _, err := resolveToolchainArg("+bad")
	require.Error(t, err)

	home := t.TempDir()
	t.Setenv(config.EnvHome, home)
	config.ResetDefaultSettingsFileCache()
	t.Cleanup(config.ResetDefaultSettingsFileCache)
	require.NoError(t, config.EnsureDirs())

	_, _, err = resolveToolchainArg("lts-1.0.5")
	require.Error(t, err)

	tcName := "lts-1.0.5"
	tcDir := filepath.Join(home, "toolchains", tcName)
	require.NoError(t, os.MkdirAll(tcDir, 0o755))
	settings := config.DefaultSettings()
	settings.DefaultToolchain = tcName
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, "settings.toml")))

	gotDir, gotName, err := resolveToolchainArg("")

	require.NoError(t, err)
	assert.Equal(t, tcDir, gotDir)
	assert.Equal(t, tcName, gotName.String())
}

func TestInstallComponentsListRollsBackPreviousComponentOnLaterFailure(t *testing.T) {
	tcName := "lts-1.0.5"
	tcDir := setupComponentCLITest(t, tcName)

	oldInstall := componentInstallFunc
	componentInstallFunc = func(ctx context.Context, roots componentlib.Roots, tc toolchain.ToolchainName, name componentlib.Name, tuple, downloadsDir string, force bool) error {
		if name == componentlib.Docs {
			return errors.New("docs failed")
		}
		require.NoError(t, os.MkdirAll(filepath.Join(roots.StdxDir, "dynamic"), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(roots.StdxDir, "dynamic", "libfoo.so"), []byte("x"), 0o644))
		return componentlib.WriteManifest(roots.TcDir, name, []string{"dynamic/libfoo.so"})
	}
	defer func() { componentInstallFunc = oldInstall }()

	err := installComponentsList(context.Background(), tcName, []string{"stdx", "docs"}, false, true)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "docs failed")
	assert.False(t, componentlib.IsInstalled(tcDir, componentlib.Stdx))
	stdxDir, dirErr := config.StdxDirFor(tcName)
	require.NoError(t, dirErr)
	assert.NoFileExists(t, filepath.Join(stdxDir, "dynamic", "libfoo.so"))
}

func TestInstallComponentsListRejectsTargetVariantToolchain(t *testing.T) {
	setupComponentCLITest(t, "lts-1.0.5-linux-x64-ohos")

	err := installComponentsList(context.Background(), "lts-1.0.5-linux-x64-ohos", []string{"stdx"}, false, true)

	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "target variant") || strings.Contains(err.Error(), "linux-x64-ohos"))
}

func TestRunComponentListQuietShowsInstalledThenAvailable(t *testing.T) {
	tcName := "lts-1.0.5"
	tcDir := setupComponentCLITest(t, tcName)
	require.NoError(t, componentlib.WriteManifest(tcDir, componentlib.Docs, []string{"index.html"}))

	oldToolchain := componentToolchain
	oldQuiet := componentListQuiet
	oldInstalledOnly := componentListInstalledOnly
	componentToolchain = tcName
	componentListQuiet = true
	componentListInstalledOnly = false
	t.Cleanup(func() {
		componentToolchain = oldToolchain
		componentListQuiet = oldQuiet
		componentListInstalledOnly = oldInstalledOnly
	})

	stdout, err := captureStdout(t, func() error {
		return runComponentList(&cobra.Command{}, nil)
	})

	require.NoError(t, err)
	lines := strings.Fields(stdout)
	assert.Equal(t, []string{"docs", "stdx", "stdx-docs"}, lines)

	componentListInstalledOnly = true
	stdout, err = captureStdout(t, func() error {
		return runComponentList(&cobra.Command{}, nil)
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"docs"}, strings.Fields(stdout))
}

func TestRunComponentAddInstallsForResolvedToolchain(t *testing.T) {
	tcName := "lts-1.0.5"
	tcDir := setupComponentCLITest(t, tcName)

	oldToolchain := componentToolchain
	oldForce := componentAddForce
	oldInstall := componentInstallFunc
	componentToolchain = tcName
	componentAddForce = true
	var gotForce bool
	componentInstallFunc = func(ctx context.Context, roots componentlib.Roots, tc toolchain.ToolchainName, name componentlib.Name, tuple, downloadsDir string, force bool) error {
		gotForce = force
		return componentlib.WriteManifest(roots.TcDir, name, []string{"index.html"})
	}
	t.Cleanup(func() {
		componentToolchain = oldToolchain
		componentAddForce = oldForce
		componentInstallFunc = oldInstall
	})

	err := runComponentAdd(&cobra.Command{}, []string{"docs"})

	require.NoError(t, err)
	assert.True(t, gotForce)
	assert.True(t, componentlib.IsInstalled(tcDir, componentlib.Docs))
}

func TestRunComponentAddRejectsCustomToolchain(t *testing.T) {
	tcName := "local-sdk"
	setupComponentCLITest(t, tcName)

	oldToolchain := componentToolchain
	componentToolchain = tcName
	t.Cleanup(func() { componentToolchain = oldToolchain })

	err := runComponentAdd(&cobra.Command{}, []string{"docs"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "docs")
}

func TestRunComponentRemoveReturnsOnlyParseErrorsWhenNoValidComponents(t *testing.T) {
	err := runComponentRemove(&cobra.Command{}, []string{"bogus", "unknown"})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "bogus")
	assert.Contains(t, err.Error(), "unknown")
}

func TestRunComponentListInstalledOnlyNoComponents(t *testing.T) {
	tcName := "lts-1.0.5"
	setupComponentCLITest(t, tcName)

	oldToolchain := componentToolchain
	oldQuiet := componentListQuiet
	oldInstalledOnly := componentListInstalledOnly
	componentToolchain = tcName
	componentListQuiet = false
	componentListInstalledOnly = true
	t.Cleanup(func() {
		componentToolchain = oldToolchain
		componentListQuiet = oldQuiet
		componentListInstalledOnly = oldInstalledOnly
	})

	stdout, err := captureStdout(t, func() error {
		return runComponentList(&cobra.Command{}, nil)
	})

	require.NoError(t, err)
	assert.NotEmpty(t, stdout)
}

func TestRunComponentListNonQuietShowsInstalledAndAvailable(t *testing.T) {
	tcName := "lts-1.0.5"
	tcDir := setupComponentCLITest(t, tcName)
	require.NoError(t, componentlib.WriteManifest(tcDir, componentlib.Docs, []string{"index.html"}))

	oldToolchain := componentToolchain
	oldQuiet := componentListQuiet
	oldInstalledOnly := componentListInstalledOnly
	componentToolchain = tcName
	componentListQuiet = false
	componentListInstalledOnly = false
	t.Cleanup(func() {
		componentToolchain = oldToolchain
		componentListQuiet = oldQuiet
		componentListInstalledOnly = oldInstalledOnly
	})

	stdout, err := captureStdout(t, func() error {
		return runComponentList(&cobra.Command{}, nil)
	})

	require.NoError(t, err)
	assert.Contains(t, stdout, "docs")
	assert.Contains(t, stdout, "stdx")
}

func TestInstallComponentsForToolchainUsesInstalledToolchain(t *testing.T) {
	tcName := "lts-1.0.5"
	tcDir := setupComponentCLITest(t, tcName)

	oldInstall := componentInstallFunc
	componentInstallFunc = func(ctx context.Context, roots componentlib.Roots, tc toolchain.ToolchainName, name componentlib.Name, tuple, downloadsDir string, force bool) error {
		return componentlib.WriteManifest(roots.TcDir, name, []string{"index.html"})
	}
	t.Cleanup(func() { componentInstallFunc = oldInstall })

	err := InstallComponentsForToolchain(context.Background(), "lts", []string{"docs"})

	require.NoError(t, err)
	assert.True(t, componentlib.IsInstalled(tcDir, componentlib.Docs))
}
