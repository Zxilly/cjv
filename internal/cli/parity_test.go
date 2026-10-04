package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/require"
)

func TestGlobalSelectorAndBatchJSON(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	for _, name := range []string{"sts-1.0.0", "lts-1.0.5"} {
		require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", name), 0755))
	}
	sf, err := config.DefaultSettingsFile()
	require.NoError(t, err)
	name := "lts-1.0.5"
	_, err = sf.Update(config.SettingsUpdate{DefaultToolchain: &name})
	require.NoError(t, err)
	text, err := executeWithOutput(t, newApplication("dev", ""), []string{"+sts-1.0.0", "--json"})
	require.NoError(t, err)
	var status rootResult
	require.NoError(t, json.Unmarshal([]byte(text), &status))
	require.Equal(t, "sts-1.0.0", status.Active)
	text, err = executeWithOutput(t, newApplication("dev", ""), []string{"uninstall", "sts-1.0", "lts-1.0.5", "--yes", "--json"})
	require.NoError(t, err)
	var removed uninstallBatchResult
	require.NoError(t, json.Unmarshal([]byte(text), &removed))
	require.Len(t, removed.Removed, 2)
}

func TestTargetManagementPinsHostAndReturnsJSON(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	server := testutil.MockServerWithTargetSDKs(t, toolchain.STS, "1.0.0", "ohos")
	sf, err := config.DefaultSettingsFile()
	require.NoError(t, err)
	url := server.URL + "/sdk-versions.json"
	_, err = sf.Update(config.SettingsUpdate{ManifestURL: &url})
	require.NoError(t, err)
	require.NoError(t, lifecycle.Install(t.Context(), lifecycle.InstallRequest{Toolchain: "sts-1.0.0"}, lifecycle.Options{}))
	text, err := executeWithOutput(t, newApplication("dev", ""), []string{"+1.0.0", "target", "add", "ohos", "--json"})
	require.NoError(t, err)
	var installed installResult
	require.NoError(t, json.Unmarshal([]byte(text), &installed))
	require.Equal(t, []string{"ohos"}, installed.Targets)
	text, err = executeWithOutput(t, newApplication("dev", ""), []string{"target", "list", "--toolchain", "sts-1.0", "--installed", "--json"})
	require.NoError(t, err)
	var targets targetListResult
	require.NoError(t, json.Unmarshal([]byte(text), &targets))
	require.Equal(t, []targetEntry{{Name: "ohos", Installed: true}}, targets.Targets)
	text, err = executeWithOutput(t, newApplication("dev", ""), []string{"+sts-1.0.0", "target", "remove", "ohos", "--json"})
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(text)))
	tuple, err := target.CurrentTargetTuple("", "ohos")
	require.NoError(t, err)
	require.NoDirExists(t, filepath.Join(home, "toolchains", "sts-1.0.0-"+tuple))
}

func TestAbsoluteSDKComponentsCannotMutateNamesake(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	external := filepath.Join(t.TempDir(), "outside-sdk")
	require.NoError(t, os.MkdirAll(external, 0755))
	roots, err := component.RootsFor("outside-sdk")
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(roots.TcDir, 0755))
	require.NoError(t, os.MkdirAll(roots.DocsDir, 0755))
	marker := filepath.Join(roots.DocsDir, "keep.html")
	require.NoError(t, os.WriteFile(marker, []byte("owned by another installation"), 0644))
	require.NoError(t, component.WriteManifest(external, component.Docs, []string{"keep.html"}))
	t.Setenv(config.EnvToolchain, external)
	_, err = executeWithOutput(t, newApplication("dev", ""), []string{"component", "remove", "docs", "--json"})
	require.ErrorContains(t, err, "external toolchain")
	require.FileExists(t, marker)
	require.True(t, component.IsInstalled(external, component.Docs))
}

func TestDefaultInstallsMissingSDKAndPreservesLegacyOffline(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	server := testutil.ValidMockServer(t)
	sf, err := config.DefaultSettingsFile()
	require.NoError(t, err)
	url := server.URL + "/sdk-versions.json"
	_, err = sf.Update(config.SettingsUpdate{ManifestURL: &url})
	require.NoError(t, err)
	text, err := executeWithOutput(t, newApplication("dev", ""), []string{"default", "lts", "--json", "--quiet"})
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(text)))
	require.DirExists(t, filepath.Join(home, "toolchains", "lts"))
	server.Close()
	legacy := filepath.Join(home, "toolchains", "sts-1.0.0")
	require.NoError(t, os.MkdirAll(legacy, 0755))
	// Simulate a published layout before the one-time migration marker.
	require.NoError(t, os.Remove(filepath.Join(home, ".cjv", "layout-v2")))
	text, err = executeWithOutput(t, newApplication("dev", ""), []string{"default", "sts", "--json"})
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(text)))
}
