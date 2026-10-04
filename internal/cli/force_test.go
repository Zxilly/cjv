package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/require"
)

func TestForceKeepsAlreadyInstalledSDKs(t *testing.T) {
	for _, operation := range []string{"install", "update"} {
		for _, identity := range []string{"sts", "sts-1.0.0"} {
			t.Run(operation+"/"+identity, func(t *testing.T) {
				home := t.TempDir()
				config.IsolateForTest(t, home)
				server := testutil.MockServerWithTargetSDKs(t, toolchain.STS, "1.0.0", "ohos")
				sf, err := config.DefaultSettingsFile()
				require.NoError(t, err)
				url := server.URL + "/sdk-versions.json"
				_, err = sf.Update(config.SettingsUpdate{ManifestURL: &url})
				require.NoError(t, err)
				require.NoError(t, lifecycle.Install(t.Context(), lifecycle.InstallRequest{Toolchain: identity, Targets: []string{"ohos"}}, lifecycle.Options{}))
				tuple, err := target.CurrentTargetTuple("", "ohos")
				require.NoError(t, err)
				for _, name := range []string{identity, identity + "-" + tuple} {
					marker := filepath.Join(home, "toolchains", name, "keep.txt")
					require.NoError(t, os.WriteFile(marker, []byte("existing SDK content"), 0o644))
				}
				_, err = executeWithOutput(t, newApplication("dev", ""), []string{operation, identity, "--force", "--json"})
				require.NoError(t, err)
				for _, name := range []string{identity, identity + "-" + tuple} {
					data, err := os.ReadFile(filepath.Join(home, "toolchains", name, "keep.txt"))
					require.NoError(t, err, "--force must not reinstall an unchanged SDK")
					require.Equal(t, "existing SDK content", string(data))
				}
			})
		}
	}
}

func TestForceAllowsMissingTargetsForInstallAndUpdate(t *testing.T) {
	for _, operation := range []string{"install", "update"} {
		t.Run(operation, func(t *testing.T) {
			home := t.TempDir()
			config.IsolateForTest(t, home)
			old := testutil.MockServerWithTargetSDKs(t, toolchain.STS, "1.0.0", "ohos")
			latest := testutil.MockServerWithTargetSDKs(t, toolchain.STS, "2.0.0")
			sf, err := config.DefaultSettingsFile()
			require.NoError(t, err)
			url := old.URL + "/sdk-versions.json"
			_, err = sf.Update(config.SettingsUpdate{ManifestURL: &url})
			require.NoError(t, err)
			require.NoError(t, lifecycle.Install(t.Context(), lifecycle.InstallRequest{Toolchain: "sts", Targets: []string{"ohos"}}, lifecycle.Options{}))
			url = latest.URL + "/sdk-versions.json"
			_, err = sf.Update(config.SettingsUpdate{ManifestURL: &url})
			require.NoError(t, err)
			_, err = executeWithOutput(t, newApplication("dev", ""), []string{operation, "sts", "--force", "--json"})
			require.NoError(t, err)
			record, err := toolchain.ReadInstallation(filepath.Join(home, "toolchains", "sts"))
			require.NoError(t, err)
			require.Equal(t, "sts-2.0.0", record.Release)
			tuple, err := target.CurrentTargetTuple("", "ohos")
			require.NoError(t, err)
			require.NoDirExists(t, filepath.Join(home, "toolchains", "sts-"+tuple))
		})
	}
}

func TestForceAllowsMissingComponentsOnDirectTargets(t *testing.T) {
	for _, fixed := range []bool{false, true} {
		t.Run(map[bool]string{false: "tracking", true: "fixed"}[fixed], func(t *testing.T) {
			home := t.TempDir()
			config.IsolateForTest(t, home)
			server := testutil.MockServerWithTargetSDKs(t, toolchain.STS, "1.0.0", "ohos")
			sf, err := config.DefaultSettingsFile()
			require.NoError(t, err)
			url := server.URL + "/sdk-versions.json"
			_, err = sf.Update(config.SettingsUpdate{ManifestURL: &url})
			require.NoError(t, err)
			host := "sts"
			if fixed {
				host = "sts-1.0.0"
			}
			require.NoError(t, lifecycle.Install(t.Context(), lifecycle.InstallRequest{Toolchain: host, Targets: []string{"ohos"}}, lifecycle.Options{}))
			tuple, err := target.CurrentTargetTuple("", "ohos")
			require.NoError(t, err)
			name := host + "-" + tuple
			marker := filepath.Join(home, "toolchains", name, "keep.txt")
			require.NoError(t, os.WriteFile(marker, []byte("existing target"), 0o644))
			before, err := os.Stat(marker)
			require.NoError(t, err)
			empty := ""
			_, err = sf.Update(config.SettingsUpdate{DefaultToolchain: &empty})
			require.NoError(t, err)
			_, err = executeWithOutput(t, newApplication("dev", ""), []string{"install", name, "--component", "docs", "--json"})
			require.Error(t, err, "fixture has no published docs")
			_, err = executeWithOutput(t, newApplication("dev", ""), []string{"install", name, "--component", "docs", "--force", "--json"})
			require.NoError(t, err)
			after, err := os.Stat(marker)
			require.NoError(t, err)
			require.True(t, os.SameFile(before, after), "a skipped new component must not replace the target SDK")
			settings, err := sf.Load()
			require.NoError(t, err)
			require.Empty(t, settings.DefaultToolchain, "a target cannot become the default")
			server.Close()
			_, err = executeWithOutput(t, newApplication("dev", ""), []string{"install", name, "--no-update", "--json"})
			require.NoError(t, err, "an unchanged target SDK must support offline --no-update")
		})
	}
}
