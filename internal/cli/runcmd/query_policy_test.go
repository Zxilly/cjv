package runcmd

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/resolve"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueriesNeverInstallMissingSDKs(t *testing.T) {
	for _, args := range [][]string{{"which", "cjc"}, {"envsetup", "--shell=bash"}, {"envsetup", "--json"}} {
		t.Run(args[0]+args[len(args)-1], func(t *testing.T) {
			home := t.TempDir()
			configtest.Isolate(t, home)
			t.Setenv(config.EnvAutoInstall, "1")
			t.Chdir(t.TempDir())
			sf, err := config.DefaultSettingsFile()
			require.NoError(t, err)
			name := "lts-1.0.5"
			_, err = sf.Update(config.SettingsUpdate{DefaultToolchain: &name})
			require.NoError(t, err)
			previous := resolve.AutoInstallFunc
			t.Cleanup(func() { resolve.AutoInstallFunc = previous })
			resolve.AutoInstallFunc = func(context.Context, string, []string) error {
				t.Fatal("a query must not install a missing SDK")
				return nil
			}
			_, err = execute(t, args...)
			var missing *cjverr.ToolchainNotInstalledError
			require.ErrorAs(t, err, &missing)
			assert.Contains(t, err.Error(), "cjv install lts-1.0.5")
			assert.NoDirExists(t, filepath.Join(home, "toolchains", name))
			assert.Equal(t, "1", os.Getenv(config.EnvAutoInstall), "query policy must not mutate the process setting")
		})
	}
}

func TestEnvsetupValidatesProjectRequirementsWithoutInstalling(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	t.Setenv(config.EnvAutoInstall, "1")
	t.Chdir(t.TempDir())
	setupFakeToolchainForCLI(t, home, "lts-1.0.5")
	require.NoError(t, os.WriteFile(config.ToolchainFileName, []byte("[toolchain]\nchannel = 'lts-1.0.5'\ncomponents = ['stdx']\n"), 0o644))
	previous := resolve.AutoInstallComponentsFunc
	t.Cleanup(func() { resolve.AutoInstallComponentsFunc = previous })
	resolve.AutoInstallComponentsFunc = func(context.Context, string, []string) error {
		t.Fatal("envsetup must not install a missing project component")
		return nil
	}
	_, err := execute(t, "envsetup", "--shell=bash")
	var missing *cjverr.ComponentNotInstalledError
	require.ErrorAs(t, err, &missing)
	assert.Equal(t, "stdx", missing.Component)
	_, err = execute(t, "which", "cjc")
	require.NoError(t, err, "which inspects an installed binary without preparing unrelated components")
}

func TestEnvsetupRejectsUnexpectedSelectors(t *testing.T) {
	configtest.Isolate(t, t.TempDir())
	for _, args := range [][]string{{"typo"}, {"+"}, {"+lts", "+sts"}, {"+bad/name"}, {"+lts", "typo"}} {
		_, err := execute(t, append([]string{"envsetup"}, args...)...)
		require.Error(t, err, "%v", args)
	}
}

func TestCustomSDKQuerySuggestsRegistration(t *testing.T) {
	configtest.Isolate(t, t.TempDir())
	t.Setenv(config.EnvToolchain, "my-sdk")
	_, err := execute(t, "which", "cjc")
	var missing *cjverr.ToolchainNotInstalledError
	require.ErrorAs(t, err, &missing)
	assert.Contains(t, err.Error(), "cjv toolchain link my-sdk <path>")
	assert.NotContains(t, err.Error(), "cjv install my-sdk")
}
