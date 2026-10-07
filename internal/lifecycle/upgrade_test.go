package lifecycle_test

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/fsops"
	"github.com/Zxilly/cjv/internal/lifecycle"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type upgradeFixture struct {
	home           string
	oldName        string
	newName        string
	sf             *config.SettingsFile
	dist           *lifecycle.InstallationDistribution
	resolved       lifecycle.ResolvedToolchain
	oldRoots       component.Roots
	newRoots       component.Roots
	initialDefault string
}

func newUpgradeFixture(t *testing.T, targetVariant, missingComponent bool) upgradeFixture {
	t.Helper()
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv(config.EnvDistServer, "")
	tuple, err := sdktarget.CurrentHostTuple("")
	require.NoError(t, err)
	oldName, newName := "lts-1.0.0", "lts-2.0.0"
	if targetVariant {
		id, err := sdktarget.ParseIdentity(tuple)
		require.NoError(t, err)
		id, err = id.WithEnvironment("ohos")
		require.NoError(t, err)
		tuple = id.Tuple()
		oldName += "-" + tuple
		newName += "-" + tuple
	}
	stdxPlatform, err := sdktarget.StdxPlatformForTuple(tuple)
	require.NoError(t, err)
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	archive := func(path string, files map[string]string) dist.ComponentInfo {
		var buf bytes.Buffer
		zw := zip.NewWriter(&buf)
		for name, data := range files {
			w, err := zw.Create(name)
			require.NoError(t, err)
			_, err = w.Write([]byte(data))
			require.NoError(t, err)
		}
		require.NoError(t, zw.Close())
		data := buf.Bytes()
		sum := sha256.Sum256(data)
		mux.HandleFunc(path, func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write(data)
			assert.NoError(t, err)
		})
		return dist.ComponentInfo{Name: filepath.Base(path), URL: server.URL + path, SHA256: hex.EncodeToString(sum[:])}
	}
	channel := dist.ChannelInfo{Latest: "2.0.0", Versions: make(map[string]map[string]dist.DownloadInfo), Components: make(map[string]dist.ComponentSet)}
	for _, version := range []string{"1.0.0", "2.0.0"} {
		sdk, hash := testutil.CreateMockSDKZip(version)
		path := "/sdk-" + version + ".zip"
		mux.HandleFunc(path, func(w http.ResponseWriter, _ *http.Request) {
			_, err := w.Write(sdk)
			assert.NoError(t, err)
		})
		channel.Versions[version] = map[string]dist.DownloadInfo{tuple: {Name: filepath.Base(path), URL: server.URL + path, SHA256: hash}}
		stdx := archive("/stdx-"+version+".zip", map[string]string{"stdx/dynamic/version.txt": version, "stdx/static/version.txt": version})
		docs := archive("/docs-"+version+".zip", map[string]string{"index.html": version})
		stdxDocs := archive("/stdx-docs-"+version+".zip", map[string]string{"index.html": version})
		set := dist.ComponentSet{Docs: &docs, StdxDocs: &stdxDocs, Stdx: map[string]dist.ComponentInfo{stdxPlatform: stdx}}
		if missingComponent && version == "2.0.0" {
			set.StdxDocs = nil
		}
		channel.Components[version] = set
	}
	manifest := dist.Manifest{}
	manifest.Channels.LTS, manifest.Channels.STS = channel, channel
	mux.HandleFunc("/versions.json", func(w http.ResponseWriter, _ *http.Request) {
		assert.NoError(t, json.NewEncoder(w).Encode(manifest))
	})
	settings := config.DefaultSettings()
	settings.ManifestURL = server.URL + "/versions.json"
	sf, err := config.DefaultSettingsFile()
	require.NoError(t, err)
	require.NoError(t, sf.Save(&settings))
	require.NoError(t, lifecycle.Install(t.Context(), lifecycle.InstallRequest{Toolchain: oldName, Components: []string{"stdx", "docs", "stdx-docs"}}, lifecycle.Options{}))
	initialDefault := oldName
	if targetVariant {
		initialDefault = "local-sdk"
		require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", initialDefault), 0o755))
	}
	_, err = sf.Update(config.SettingsUpdate{DefaultToolchain: &initialDefault, Overrides: map[string]string{filepath.Join(home, "project"): initialDefault}})
	require.NoError(t, err)
	d, err := lifecycle.OpenInstallationDistribution(t.Context(), lifecycle.Options{})
	require.NoError(t, err)
	resolved, err := d.Resolve(t.Context(), toolchain.ToolchainName{Channel: toolchain.LTS}, tuple)
	require.NoError(t, err)
	oldRoots, err := component.RootsFor(oldName)
	require.NoError(t, err)
	newRoots, err := component.RootsFor(newName)
	require.NoError(t, err)
	return upgradeFixture{home, oldName, newName, sf, d, resolved, oldRoots, newRoots, initialDefault}
}

func linkUpgradeStdx(t *testing.T, roots component.Roots, content string) string {
	t.Helper()
	source := t.TempDir()
	for _, child := range []string{"dynamic", "static"} {
		require.NoError(t, os.MkdirAll(filepath.Join(source, child), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(source, child, "user.txt"), []byte(content), 0o644))
	}
	_, err := component.Link(roots, component.Stdx, source, true)
	require.NoError(t, err)
	return source
}

func TestRemoveFailureRestoresComponentRootsAndReferences(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows working-directory handle prevents SDK retirement")
	}
	f := newUpgradeFixture(t, false, false)
	before, err := os.ReadFile(f.sf.Path())
	require.NoError(t, err)
	t.Chdir(f.oldRoots.TcDir)
	err = lifecycle.RemoveToolchain(f.oldName)
	require.Error(t, err)
	for _, path := range []string{f.oldRoots.TcDir, f.oldRoots.StdxDir, f.oldRoots.DocsDir} {
		assert.DirExists(t, path)
	}
	after, err := os.ReadFile(f.sf.Path())
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestForceInstallKeepsExternalComponentsManaged(t *testing.T) {
	f := newUpgradeFixture(t, false, false)
	source := linkUpgradeStdx(t, f.oldRoots, "my libraries")
	require.NoError(t, lifecycle.Install(t.Context(), lifecycle.InstallRequest{Toolchain: f.oldName, Force: true}, lifecycle.Options{}))
	intents, err := component.InstalledIntents(f.oldRoots)
	require.NoError(t, err)
	assert.Len(t, intents, 3)
	assert.Contains(t, intents, component.Intent{Name: component.Stdx, Source: source})
	require.NoError(t, lifecycle.RemoveToolchain(f.oldName))
	assert.NoDirExists(t, f.oldRoots.DocsDir)
	assert.NoDirExists(t, f.oldRoots.StdxDir)
	assert.FileExists(t, filepath.Join(source, "dynamic", "user.txt"))
}

func fixtureChannel(t *testing.T, f upgradeFixture) component.Roots {
	t.Helper()
	name, err := toolchain.ParseToolchainName(f.oldName)
	require.NoError(t, err)
	name.Version = ""
	roots, err := component.RootsFor(name.String())
	require.NoError(t, err)
	for _, pair := range [][2]string{{f.oldRoots.TcDir, roots.TcDir}, {f.oldRoots.DocsDir, roots.DocsDir}, {f.oldRoots.StdxDir, roots.StdxDir}} {
		require.NoError(t, fsops.CopyTree(pair[0], pair[1]))
	}
	return roots
}
func TestChannelUpgradePublishesAllRootsAndKeepsFixedVersion(t *testing.T) {
	for _, target := range []bool{false, true} {
		t.Run(fmt.Sprint(target), func(t *testing.T) {
			f := newUpgradeFixture(t, target, false)
			channel := fixtureChannel(t, f)
			_, err := lifecycle.UpgradeToolchain(t.Context(), filepath.Base(channel.TcDir), f.resolved, f.dist, lifecycle.Options{})
			require.NoError(t, err)
			installed, err := toolchain.ReadInstallation(channel.TcDir)
			require.NoError(t, err)
			assert.Equal(t, f.newName, installed.Release)
			for _, path := range []string{filepath.Join(channel.StdxDir, "dynamic", "version.txt"), filepath.Join(channel.DocsDir, "main", "index.html"), filepath.Join(channel.DocsDir, "stdx", "index.html")} {
				data, err := os.ReadFile(path)
				require.NoError(t, err)
				assert.Equal(t, "2.0.0", string(data))
			}
			old, err := os.ReadFile(filepath.Join(f.oldRoots.StdxDir, "dynamic", "version.txt"))
			require.NoError(t, err)
			assert.Equal(t, "1.0.0", string(old))
			settings, err := f.sf.Load()
			require.NoError(t, err)
			assert.Equal(t, f.initialDefault, settings.DefaultToolchain)
		})
	}
}
func TestChannelUpgradeFailureRetainsCompleteOldInstallation(t *testing.T) {
	for _, fail := range []string{"component", "finalize"} {
		t.Run(fail, func(t *testing.T) {
			f := newUpgradeFixture(t, false, fail == "component")
			channel := fixtureChannel(t, f)
			if fail == "finalize" {
				lifecycle.SetAfterFinalizeHook(t, func() error { return errors.New("finalize failed") })
			}
			_, err := lifecycle.UpgradeToolchain(t.Context(), "lts", f.resolved, f.dist, lifecycle.Options{})
			require.Error(t, err)
			installed, err := toolchain.ReadInstallation(channel.TcDir)
			require.NoError(t, err)
			assert.Equal(t, f.oldName, installed.Release)
			for _, path := range []string{filepath.Join(channel.StdxDir, "dynamic", "version.txt"), filepath.Join(channel.DocsDir, "stdx", "index.html")} {
				data, err := os.ReadFile(path)
				require.NoError(t, err)
				assert.Equal(t, "1.0.0", string(data))
			}
			assert.NoDirExists(t, f.newRoots.TcDir)
		})
	}
}
func TestChannelComponentChoicesDoNotAffectFixedInstall(t *testing.T) {
	f := newUpgradeFixture(t, false, false)
	channel := fixtureChannel(t, f)
	require.NoError(t, component.Remove(channel, component.Docs))
	assert.True(t, component.IsInstalled(f.oldRoots.TcDir, component.Docs))
	assert.False(t, component.IsInstalled(channel.TcDir, component.Docs))
	source := linkUpgradeStdx(t, channel, "user libraries")
	_, err := lifecycle.UpgradeToolchain(t.Context(), "lts", f.resolved, f.dist, lifecycle.Options{})
	require.NoError(t, err)
	intents, err := component.InstalledIntents(channel)
	require.NoError(t, err)
	assert.Contains(t, intents, component.Intent{Name: component.Stdx, Source: source})
	assert.False(t, component.IsInstalled(channel.TcDir, component.Docs))
	require.NoError(t, lifecycle.RemoveToolchain("lts"))
	assert.FileExists(t, filepath.Join(source, "dynamic", "user.txt"))
	assert.DirExists(t, f.oldRoots.TcDir)
}

func TestChannelUpgradeRecoversRealCrashWithAllComponents(t *testing.T) {
	if os.Getenv("CJV_TEST_CHANNEL_CRASH") == "1" {
		config.IsolateForTest(t, os.Getenv(config.EnvHome))
		lifecycle.SetAfterFinalizeHook(t, func() error { os.Exit(73); return nil })
		_, err := lifecycle.UpdateInstalled(t.Context(), toolchain.ToolchainName{Channel: toolchain.LTS}, lifecycle.Options{})
		t.Fatalf("expected interruption, got %v", err)
	}
	f := newUpgradeFixture(t, false, false)
	channel := fixtureChannel(t, f)
	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestChannelUpgradeRecoversRealCrashWithAllComponents$")
	cmd.Env = append(os.Environ(), "CJV_TEST_CHANNEL_CRASH=1")
	output, err := cmd.CombinedOutput()
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr, "%s", output)
	require.Equal(t, 73, exitErr.ExitCode(), "%s", output)
	record, err := toolchain.ReadInstallation(channel.TcDir)
	require.NoError(t, err)
	assert.Equal(t, f.newName, record.Release, "crash occurs after every new root has been placed")
	require.NoError(t, toolchain.RecoverHome())
	record, err = toolchain.ReadInstallation(channel.TcDir)
	require.NoError(t, err)
	assert.Equal(t, f.oldName, record.Release)
	for _, path := range []string{filepath.Join(channel.StdxDir, "dynamic", "version.txt"), filepath.Join(channel.DocsDir, "main", "index.html"), filepath.Join(channel.DocsDir, "stdx", "index.html")} {
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.Equal(t, "1.0.0", string(data))
	}
	assert.FileExists(t, compilerPath(f.oldRoots.TcDir))
}
