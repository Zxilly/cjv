package lifecycle_test

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/fstx"
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
	fetcher        *lifecycle.ManifestFetcher
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
	tuple, err := dist.CurrentHostTuple("")
	require.NoError(t, err)
	oldName, newName := "lts-1.0.0", "lts-2.0.0"
	if targetVariant {
		tuple, err = sdktarget.BuildTuple(tuple, "ohos")
		require.NoError(t, err)
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
	require.NoError(t, lifecycle.InstallToolchainWithExtras(t.Context(), oldName, nil, []string{"stdx", "docs", "stdx-docs"}, false, lifecycle.Options{}))
	initialDefault := oldName
	if targetVariant {
		initialDefault = "local-sdk"
		require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", initialDefault), 0o755))
	}
	_, err = sf.Update(config.SettingsUpdate{DefaultToolchain: &initialDefault, Overrides: map[string]string{filepath.Join(home, "project"): initialDefault}})
	require.NoError(t, err)
	settingsPtr, err := sf.Load()
	require.NoError(t, err)
	fetcher, err := lifecycle.NewManifestFetcherForSettings(settingsPtr, lifecycle.Options{})
	require.NoError(t, err)
	resolved, err := lifecycle.ResolveAndLocatePlatform(t.Context(), toolchain.ToolchainName{Channel: toolchain.LTS}, settingsPtr, fetcher, tuple)
	require.NoError(t, err)
	oldRoots, err := component.RootsFor(oldName)
	require.NoError(t, err)
	newRoots, err := component.RootsFor(newName)
	require.NoError(t, err)
	return upgradeFixture{home, oldName, newName, sf, fetcher, resolved, oldRoots, newRoots, initialDefault}
}

func (f upgradeFixture) upgrade(t *testing.T, opts lifecycle.Options) (bool, error) {
	t.Helper()
	return lifecycle.UpgradeToolchain(t.Context(), f.oldName, f.resolved, f.sf, f.fetcher, opts)
}

func TestUpgradeMigratesComponentsAndRetiresAllOldRoots(t *testing.T) {
	for _, targetVariant := range []bool{false, true} {
		name := "host"
		if targetVariant {
			name = "target"
		}
		t.Run(name, func(t *testing.T) {
			f := newUpgradeFixture(t, targetVariant, false)
			updated, err := f.upgrade(t, lifecycle.Options{})
			require.NoError(t, err)
			assert.True(t, updated)
			for _, path := range []string{f.oldRoots.TcDir, f.oldRoots.StdxDir, f.oldRoots.DocsDir} {
				assert.NoDirExists(t, path)
			}
			components, err := component.ListInstalled(f.newRoots.TcDir)
			require.NoError(t, err)
			assert.ElementsMatch(t, component.KnownComponents(), components)
			for _, path := range []string{filepath.Join(f.newRoots.StdxDir, "dynamic", "version.txt"), filepath.Join(f.newRoots.DocsDir, "main", "index.html"), filepath.Join(f.newRoots.DocsDir, "stdx", "index.html")} {
				data, err := os.ReadFile(path)
				require.NoError(t, err)
				assert.Equal(t, "2.0.0", string(data))
			}
			settings, err := f.sf.Load()
			require.NoError(t, err)
			expectedDefault := f.newName
			if targetVariant {
				expectedDefault = f.initialDefault
			}
			assert.Equal(t, expectedDefault, settings.DefaultToolchain)
			assert.Equal(t, expectedDefault, settings.Overrides[filepath.Join(f.home, "project")])
		})
	}
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

func TestUpgradePreservesLinkedSourceAndExistingReplacementChoices(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(map[bool]string{false: "new", true: "existing"}[existing], func(t *testing.T) {
			f := newUpgradeFixture(t, false, false)
			oldSource := linkUpgradeStdx(t, f.oldRoots, "old user source")
			selectedSource := oldSource
			if existing {
				settings, err := f.sf.Load()
				require.NoError(t, err)
				require.NoError(t, lifecycle.InstallResolvedNoDefault(t.Context(), f.resolved, settings, f.sf, false, lifecycle.Options{}))
				selectedSource = linkUpgradeStdx(t, f.newRoots, "new user choice")
			}
			_, err := f.upgrade(t, lifecycle.Options{})
			require.NoError(t, err)
			intents, err := component.InstalledIntents(f.newRoots)
			require.NoError(t, err)
			assert.Contains(t, intents, component.Intent{Name: component.Stdx, Source: selectedSource})
			assert.FileExists(t, filepath.Join(oldSource, "dynamic", "user.txt"))
			require.NoError(t, lifecycle.RemoveToolchain(f.newName))
			assert.FileExists(t, filepath.Join(oldSource, "dynamic", "user.txt"))
			assert.FileExists(t, filepath.Join(selectedSource, "static", "user.txt"))
			assert.NoDirExists(t, f.newRoots.StdxDir)
			assert.NoDirExists(t, f.newRoots.DocsDir)
		})
	}
}

func TestUpgradeMissingComponentLeavesOldVersionUsable(t *testing.T) {
	f := newUpgradeFixture(t, false, true)
	before, err := os.ReadFile(f.sf.Path())
	require.NoError(t, err)
	updated, err := f.upgrade(t, lifecycle.Options{})
	require.Error(t, err)
	assert.False(t, updated)
	for _, path := range []string{f.oldRoots.TcDir, f.oldRoots.StdxDir, f.oldRoots.DocsDir} {
		assert.DirExists(t, path)
	}
	components, err := component.ListInstalled(f.newRoots.TcDir)
	require.NoError(t, err)
	assert.Empty(t, components, "partially added replacement components must be undone")
	assert.NoDirExists(t, f.newRoots.TcDir, "channel resolution must keep selecting the old usable version")
	selected, err := toolchain.FindInstalled(toolchain.ToolchainName{Channel: toolchain.LTS})
	require.NoError(t, err)
	assert.Equal(t, f.oldRoots.TcDir, selected)
	after, err := os.ReadFile(f.sf.Path())
	require.NoError(t, err)
	assert.Equal(t, before, after)
}

func TestFailedUpgradePreservesExistingReplacementChoices(t *testing.T) {
	f := newUpgradeFixture(t, false, true)
	settings, err := f.sf.Load()
	require.NoError(t, err)
	require.NoError(t, lifecycle.InstallResolvedNoDefault(t.Context(), f.resolved, settings, f.sf, false, lifecycle.Options{}))
	source := linkUpgradeStdx(t, f.newRoots, "existing destination choice")
	_, err = f.upgrade(t, lifecycle.Options{})
	require.Error(t, err)
	assert.DirExists(t, f.oldRoots.TcDir)
	assert.DirExists(t, f.newRoots.TcDir)
	intents, err := component.InstalledIntents(f.newRoots)
	require.NoError(t, err)
	assert.Equal(t, []component.Intent{{Name: component.Stdx, Source: source}}, intents)
	assert.FileExists(t, filepath.Join(source, "dynamic", "user.txt"))
	settings, err = f.sf.Load()
	require.NoError(t, err)
	assert.Equal(t, f.oldName, settings.DefaultToolchain)
}

func TestUpgradeReportsIncompleteReplacementCleanupFailure(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows working-directory handle prevents replacement cleanup")
	}
	f := newUpgradeFixture(t, false, true)
	lifecycle.SetAfterFinalizeHook(t, func() error {
		t.Chdir(f.newRoots.TcDir)
		return nil
	})
	_, err := f.upgrade(t, lifecycle.Options{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "remove incomplete replacement")
	assert.ErrorContains(t, err, "stdx-docs")
	installed, err := toolchain.ListInstalled()
	require.NoError(t, err)
	assert.Contains(t, installed, f.oldName)
	assert.Contains(t, installed, f.newName, "obstructed replacement must remain discoverable")
	for _, path := range []string{f.oldRoots.TcDir, f.oldRoots.StdxDir, f.oldRoots.DocsDir} {
		assert.DirExists(t, path)
	}
}

func TestUpgradeSettingsFailureRestoresOldContent(t *testing.T) {
	f := newUpgradeFixture(t, false, false)
	before, err := os.ReadFile(f.sf.Path())
	require.NoError(t, err)
	lifecycle.SetAfterFinalizeHook(t, func() error {
		// Block the real settings write after materializing the replacement.
		require.NoError(t, os.Rename(f.sf.Path(), f.sf.Path()+".saved"))
		require.NoError(t, os.Mkdir(f.sf.Path(), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(f.sf.Path(), "obstruction"), []byte("keep"), 0o644))
		return nil
	})
	_, err = f.upgrade(t, lifecycle.Options{})
	require.Error(t, err)
	for _, path := range []string{f.oldRoots.TcDir, f.oldRoots.StdxDir, f.oldRoots.DocsDir} {
		assert.DirExists(t, path)
	}
	data, err := os.ReadFile(filepath.Join(f.oldRoots.StdxDir, "dynamic", "version.txt"))
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", string(data))
	unchanged, err := os.ReadFile(f.sf.Path() + ".saved")
	require.NoError(t, err)
	assert.Equal(t, before, unchanged)
	components, err := component.ListInstalled(f.newRoots.TcDir)
	require.NoError(t, err)
	assert.ElementsMatch(t, component.KnownComponents(), components,
		"unreadable settings must retain the complete usable replacement")
}

func TestRemovalAndUpgradeRecoverBeforeInspectingOldContent(t *testing.T) {
	for _, operation := range []string{"remove", "upgrade"} {
		t.Run(operation, func(t *testing.T) {
			f := newUpgradeFixture(t, false, false)
			tx, err := fstx.NewToolchainTransaction(f.home, f.oldName)
			require.NoError(t, err)
			for _, path := range []string{f.oldRoots.DocsDir, f.oldRoots.StdxDir, f.oldRoots.TcDir} {
				require.NoError(t, tx.RemoveDir(path))
			}
			// Discard the unfinished transaction as an interrupted process
			// would. The production operation must recover before inspecting.
			assert.NoDirExists(t, f.oldRoots.TcDir)
			if operation == "remove" {
				require.NoError(t, lifecycle.PrepareToolchainRemoval(f.oldName))
				for _, path := range []string{f.oldRoots.TcDir, f.oldRoots.StdxDir, f.oldRoots.DocsDir} {
					assert.DirExists(t, path, "source must be inspectable before confirmation")
				}
				require.NoError(t, lifecycle.RemoveToolchain(f.oldName))
			} else {
				_, err := f.upgrade(t, lifecycle.Options{})
				require.NoError(t, err)
				components, err := component.ListInstalled(f.newRoots.TcDir)
				require.NoError(t, err)
				assert.ElementsMatch(t, component.KnownComponents(), components)
			}
			for _, path := range []string{f.oldRoots.TcDir, f.oldRoots.StdxDir, f.oldRoots.DocsDir} {
				assert.NoDirExists(t, path)
			}
		})
	}
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

func TestForceReinstallKeepsExternalComponentsManaged(t *testing.T) {
	f := newUpgradeFixture(t, false, false)
	source := linkUpgradeStdx(t, f.oldRoots, "my libraries")
	require.NoError(t, lifecycle.InstallToolchainWithExtras(t.Context(), f.oldName, nil, nil, true, lifecycle.Options{}))
	intents, err := component.InstalledIntents(f.oldRoots)
	require.NoError(t, err)
	assert.Len(t, intents, 3)
	assert.Contains(t, intents, component.Intent{Name: component.Stdx, Source: source})
	require.NoError(t, lifecycle.RemoveToolchain(f.oldName))
	assert.NoDirExists(t, f.oldRoots.DocsDir)
	assert.NoDirExists(t, f.oldRoots.StdxDir)
	assert.FileExists(t, filepath.Join(source, "dynamic", "user.txt"))
}
