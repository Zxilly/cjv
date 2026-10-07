package lifecycle

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/fsops"
	"github.com/Zxilly/cjv/internal/fstx"
	"github.com/Zxilly/cjv/internal/progress"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const publicationToolchain = "lts-1.0.5"

func componentPublicationFixture(t *testing.T) (component.Roots, string) {
	t.Helper()
	installedToolchainHome(t, publicationToolchain)
	roots, err := component.RootsFor(publicationToolchain)
	require.NoError(t, err)
	user := t.TempDir()
	for _, sub := range []string{"dynamic", "static"} {
		require.NoError(t, os.MkdirAll(filepath.Join(user, sub), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(user, sub, "old"), []byte("user"), 0o644))
	}
	_, err = component.Link(roots, component.Stdx, user, false)
	require.NoError(t, err)
	for _, name := range []component.Name{component.Docs, component.StdxDocs} {
		spec, err := component.SpecFor(name)
		require.NoError(t, err)
		dir := spec.InstallRoot(roots)
		require.NoError(t, os.MkdirAll(dir, 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "old.html"), []byte("old"), 0o644))
		require.NoError(t, os.WriteFile(filepath.Join(dir, "untracked"), []byte("user"), 0o644))
		require.NoError(t, component.WriteManifest(roots.TcDir, name, []string{"old.html"}))
	}
	require.NoError(t, os.WriteFile(filepath.Join(roots.TcDir, "sdk-payload"), []byte("SDK"), 0o644))
	componentServer(t, map[string]string{"new.html": "new"},
		map[string]string{"top/dynamic/new": "new", "top/static/new": "new"}, []string{hostStdxPlatform(t)})
	// Finish migration before observing publication; this test is about one
	// fixed installation, not startup materialization of its tracking identity.
	require.NoError(t, toolchain.RecoverHome())
	return roots, user
}

func assertOriginalComponents(t *testing.T, roots component.Roots, user string) {
	t.Helper()
	intents, err := component.InstalledIntents(roots)
	require.NoError(t, err)
	for _, intent := range intents {
		if intent.Name == component.Stdx {
			assert.Equal(t, user, intent.Source)
		}
	}
	for _, sub := range []string{"dynamic", "static"} {
		assert.FileExists(t, filepath.Join(roots.StdxDir, sub, "old"))
		assert.NoFileExists(t, filepath.Join(roots.StdxDir, sub, "new"))
		assert.FileExists(t, filepath.Join(user, sub, "old"))
		assert.NoFileExists(t, filepath.Join(user, sub, "new"))
	}
	assert.FileExists(t, filepath.Join(roots.DocsDir, "main", "old.html"))
	assert.NoFileExists(t, filepath.Join(roots.DocsDir, "main", "new.html"))
	paths, err := component.ReadManifest(roots.TcDir, component.Docs)
	require.NoError(t, err)
	assert.Equal(t, []string{"old.html"}, paths)
	assertUntouchedComponents(t, roots)
}

func assertUntouchedComponents(t *testing.T, roots component.Roots) {
	t.Helper()
	assert.FileExists(t, filepath.Join(roots.TcDir, "sdk-payload"))
	assert.FileExists(t, filepath.Join(roots.DocsDir, "main", "untracked"))
	assert.FileExists(t, filepath.Join(roots.DocsDir, "stdx", "untracked"))
	assert.FileExists(t, filepath.Join(roots.DocsDir, "stdx", "old.html"))
	paths, err := component.ReadManifest(roots.TcDir, component.StdxDocs)
	require.NoError(t, err)
	assert.Equal(t, []string{"old.html"}, paths)
}

func TestComponentBatchRecoversRealProcessInterruption(t *testing.T) {
	if phase := os.Getenv("CJV_TEST_COMPONENT_CRASH"); phase != "" {
		config.IsolateForTest(t, os.Getenv(config.EnvHome))
		afterComponentRootHook = func(path string) error {
			if phase == "first" || filepath.Base(path) == "components" {
				os.Exit(73)
			}
			return nil
		}
		_ = InstallComponents(t.Context(), publicationToolchain, []string{"stdx", "docs"}, true, Options{})
		os.Exit(74)
	}
	for _, phase := range []string{"first", "metadata"} {
		t.Run(phase, func(t *testing.T) {
			roots, user := componentPublicationFixture(t)
			cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestComponentBatchRecoversRealProcessInterruption$")
			cmd.Env = append(os.Environ(), "CJV_TEST_COMPONENT_CRASH="+phase)
			output, err := cmd.CombinedOutput()
			var exitErr *exec.ExitError
			require.ErrorAs(t, err, &exitErr, "%s", output)
			require.Equal(t, 73, exitErr.ExitCode(), "%s", output)
			assert.FileExists(t, filepath.Join(roots.StdxDir, "dynamic", "new"))
			if phase == "metadata" {
				paths, err := component.ReadManifest(roots.TcDir, component.Docs)
				require.NoError(t, err)
				assert.Equal(t, []string{"new.html"}, paths)
			}
			require.NoError(t, toolchain.RecoverHome())
			assertOriginalComponents(t, roots, user)
			require.NoError(t, toolchain.RecoverHome(), "recovery must be idempotent")
		})
	}
}

func TestComponentBatchPublicationRollbackRetainsRecoverableBackup(t *testing.T) {
	for _, obstruct := range []bool{false, true} {
		t.Run(fmt.Sprint(obstruct), func(t *testing.T) {
			roots, user := componentPublicationFixture(t)
			failure := errors.New("publication interrupted")
			stage := config.StagingDir(roots.StdxDir)
			afterComponentRootHook = func(string) error {
				assert.NoFileExists(t, filepath.Join(config.StagingDir(roots.TcDir), "sdk-payload"), "component publication must not stage the SDK payload")
				if obstruct {
					require.NoError(t, os.MkdirAll(stage, 0o755))
				}
				return failure
			}
			t.Cleanup(func() { afterComponentRootHook = nil })
			installedEvents := 0
			opts := Options{Progress: componentInstallSink(func(e progress.Event) {
				if e.Kind == progress.ComponentInstalled {
					installedEvents++
				}
			})}
			err := InstallComponents(t.Context(), publicationToolchain, []string{"stdx", "docs"}, true, opts)
			require.ErrorIs(t, err, failure)
			assert.Zero(t, installedEvents, "success events belong after the batch commit")
			if obstruct {
				var recovery *fstx.RecoveryError
				require.ErrorAs(t, err, &recovery)
				assert.DirExists(t, recovery.Directory)
				assert.DirExists(t, config.StagingDir(roots.TcDir), "metadata staging is retained until rollback succeeds")
				require.Error(t, toolchain.RecoverHome(), "obstruction must retain the journal again")
				require.NoError(t, os.Remove(stage))
				require.NoError(t, toolchain.RecoverHome())
				assert.NoDirExists(t, recovery.Directory)
			}
			assertOriginalComponents(t, roots, user)
		})
	}
}

func TestComponentBatchCommitPreservesSDKAndUnselectedDocs(t *testing.T) {
	roots, user := componentPublicationFixture(t)
	before, err := os.Stat(filepath.Join(roots.TcDir, "sdk-payload"))
	require.NoError(t, err)
	require.NoError(t, InstallComponents(t.Context(), publicationToolchain, []string{"stdx", "docs"}, true, Options{}))
	assertUntouchedComponents(t, roots)
	after, err := os.Stat(filepath.Join(roots.TcDir, "sdk-payload"))
	require.NoError(t, err)
	assert.True(t, os.SameFile(before, after))
	assert.FileExists(t, filepath.Join(roots.StdxDir, "dynamic", "new"))
	assert.FileExists(t, filepath.Join(roots.DocsDir, "main", "new.html"))
	assert.NoFileExists(t, filepath.Join(roots.DocsDir, "main", "old.html"))
	assert.FileExists(t, filepath.Join(user, "dynamic", "old"))
	assert.NoFileExists(t, filepath.Join(user, "dynamic", "new"))
	require.NoError(t, toolchain.RecoverHome())
	assert.FileExists(t, filepath.Join(roots.DocsDir, "main", "new.html"))
}

func TestComponentBatchRefusesToMergeThroughUntrackedUserLink(t *testing.T) {
	roots, user := componentPublicationFixture(t)
	linked := filepath.Join(roots.DocsDir, "main", "linked")
	require.NoError(t, fsops.SymlinkOrJunction(user, linked))
	componentServer(t, map[string]string{"linked/new.html": "must stay private"}, nil, nil)
	err := InstallComponents(t.Context(), publicationToolchain, []string{"docs"}, true, Options{})
	require.ErrorContains(t, err, "would traverse untracked symlink")
	assert.NoFileExists(t, filepath.Join(user, "new.html"))
	link, err := os.Lstat(linked)
	require.NoError(t, err)
	assert.NotZero(t, link.Mode()&os.ModeSymlink, "the live user link stays intact")
	assertOriginalComponents(t, roots, user)
}

func TestComponentBatchRefusesToRemoveThroughManifestParentLink(t *testing.T) {
	roots, user := componentPublicationFixture(t)
	require.NoError(t, os.WriteFile(filepath.Join(user, "old.html"), []byte("user-owned"), 0o644))
	linked := filepath.Join(roots.DocsDir, "main", "linked")
	require.NoError(t, fsops.SymlinkOrJunction(user, linked))
	require.NoError(t, component.WriteManifest(roots.TcDir, component.Docs, []string{"linked/old.html"}))
	err := InstallComponents(t.Context(), publicationToolchain, []string{"docs"}, true, Options{})
	require.ErrorContains(t, err, "would traverse manifest parent symlink")
	data, err := os.ReadFile(filepath.Join(user, "old.html"))
	require.NoError(t, err)
	assert.Equal(t, "user-owned", string(data))
	paths, err := component.ReadManifest(roots.TcDir, component.Docs)
	require.NoError(t, err)
	assert.Equal(t, []string{"linked/old.html"}, paths)
	assertUntouchedComponents(t, roots)
}

func TestComponentBatchRejectsEscapingManifestPaths(t *testing.T) {
	for _, path := range []string{"../stdx/old.html", "."} {
		t.Run(path, func(t *testing.T) {
			roots, _ := componentPublicationFixture(t)
			require.NoError(t, component.WriteManifest(roots.TcDir, component.Docs, []string{path}))
			err := InstallComponents(t.Context(), publicationToolchain, []string{"docs"}, true, Options{})
			require.ErrorContains(t, err, "manifest path escapes its root")
			assertUntouchedComponents(t, roots)
			paths, err := component.ReadManifest(roots.TcDir, component.Docs)
			require.NoError(t, err)
			assert.Equal(t, []string{path}, paths)
		})
	}
}
