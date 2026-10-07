package component

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/fsops"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComponentEditsRejectUnsafeOwnershipBeforeMutation(t *testing.T) {
	for _, operation := range []string{"remove", "link", "archive"} {
		for _, damage := range []string{"escape", "parent-link", "root-link", "metadata-link"} {
			t.Run(operation+"/"+damage, func(t *testing.T) {
				base := t.TempDir()
				roots := Roots{TcDir: filepath.Join(base, "sdk"), StdxDir: filepath.Join(base, "stdx"), DocsDir: filepath.Join(base, "docs")}
				user := makeStdxSource(t, "user-owned")
				require.NoError(t, os.MkdirAll(filepath.Join(roots.StdxDir, "dynamic"), 0o755))
				owned := filepath.Join(roots.StdxDir, "owned")
				require.NoError(t, os.WriteFile(owned, []byte("old"), 0o644))
				paths := []string{"owned"}
				switch damage {
				case "escape":
					require.NoError(t, os.WriteFile(filepath.Join(base, "outside"), []byte("user-owned"), 0o644))
					paths = append(paths, "../outside")
				case "parent-link":
					require.NoError(t, os.Remove(filepath.Join(roots.StdxDir, "dynamic")))
					require.NoError(t, fsops.SymlinkOrJunction(filepath.Join(user, "dynamic"), filepath.Join(roots.StdxDir, "dynamic")))
					paths = append(paths, "dynamic/user-owned")
				case "root-link":
					require.NoError(t, os.Remove(owned))
					require.NoError(t, os.Remove(filepath.Join(roots.StdxDir, "dynamic")))
					require.NoError(t, os.Remove(roots.StdxDir))
					require.NoError(t, fsops.SymlinkOrJunction(user, roots.StdxDir))
					paths = []string{"dynamic/user-owned"}
				case "metadata-link":
					require.NoError(t, os.MkdirAll(roots.TcDir, 0o755))
					require.NoError(t, fsops.SymlinkOrJunction(t.TempDir(), filepath.Join(roots.TcDir, ".cjv")))
				}
				require.NoError(t, WriteManifest(roots.TcDir, Stdx, paths))
				before, err := os.Stat(owned)
				if damage != "root-link" {
					require.NoError(t, err)
				}
				var editErr error
				switch operation {
				case "remove":
					editErr = Remove(roots, Stdx)
				case "link":
					_, editErr = Link(roots, Stdx, makeStdxSource(t, "new"), true)
				case "archive":
					archive := buildZip(t, t.TempDir(), "stdx.zip", map[string]string{"stdx/dynamic/new": "new", "stdx/static/new": "new"})
					editErr = InstallFromArchive(t.Context(), roots, Stdx, archive, true)
				}
				require.Error(t, editErr)
				assert.FileExists(t, filepath.Join(user, "dynamic", "user-owned"))
				assert.NoFileExists(t, filepath.Join(user, "dynamic", "new"))
				if damage == "escape" {
					assert.FileExists(t, filepath.Join(base, "outside"))
				}
				if damage != "root-link" {
					after, statErr := os.Stat(owned)
					require.NoError(t, statErr)
					assert.True(t, os.SameFile(before, after), "preflight rejection must not replace live files through snapshot restoration")
				}
				got, err := ReadManifest(roots.TcDir, Stdx)
				require.NoError(t, err)
				assert.Equal(t, paths, got)
			})
		}
	}
}

func TestArchiveRefusesUntrackedDirectoryLinkBeforeMutation(t *testing.T) {
	for _, installed := range []bool{false, true} {
		t.Run(map[bool]string{false: "first", true: "replacement"}[installed], func(t *testing.T) {
			roots := linkRoots(t)
			user := makeStdxSource(t, "user-owned")
			require.NoError(t, fsops.SymlinkOrJunction(filepath.Join(user, "dynamic"), filepath.Join(roots.StdxDir, "dynamic")))
			owned := filepath.Join(roots.StdxDir, "a-owned")
			require.NoError(t, os.WriteFile(owned, []byte("old"), 0o644))
			if installed {
				require.NoError(t, WriteManifest(roots.TcDir, Stdx, []string{"a-owned"}))
			}
			before, err := os.Stat(owned)
			require.NoError(t, err)
			archive := buildZip(t, t.TempDir(), "stdx.zip", map[string]string{"stdx/a-owned": "new", "stdx/dynamic/new": "new"})
			err = InstallFromArchive(t.Context(), roots, Stdx, archive, installed)
			require.ErrorContains(t, err, "would traverse untracked symlink")
			assert.NoFileExists(t, filepath.Join(user, "dynamic", "new"))
			after, err := os.Stat(owned)
			require.NoError(t, err)
			assert.True(t, os.SameFile(before, after))
		})
	}
}

func TestComponentRestoreRefusesRedirectedRootsAndRetainsBackup(t *testing.T) {
	for _, redirect := range []string{"root", "metadata", "parent"} {
		t.Run(redirect, func(t *testing.T) {
			roots := linkRoots(t)
			if redirect == "parent" {
				roots.StdxDir = filepath.Join(t.TempDir(), "components", "stdx")
				require.NoError(t, os.MkdirAll(roots.StdxDir, 0o755))
			}
			owned := filepath.Join(roots.StdxDir, "old")
			require.NoError(t, os.WriteFile(owned, []byte("old"), 0o644))
			require.NoError(t, WriteManifest(roots.TcDir, Stdx, []string{"old"}))
			user := t.TempDir()
			if redirect == "parent" {
				require.NoError(t, os.MkdirAll(filepath.Join(user, "stdx"), 0o755))
				require.NoError(t, os.WriteFile(filepath.Join(user, "stdx", "old"), []byte("external"), 0o644))
			}
			userFile := filepath.Join(user, "user-owned")
			require.NoError(t, os.WriteFile(userFile, []byte("user"), 0o644))
			before, err := os.Stat(userFile)
			require.NoError(t, err)
			failure := errors.New("apply failed after redirection")
			err = ApplyChanges(roots, []Name{Stdx}, func() error {
				path := roots.StdxDir
				switch redirect {
				case "metadata":
					path = filepath.Join(roots.TcDir, ".cjv")
				case "parent":
					path = filepath.Dir(roots.StdxDir)
				}
				require.NoError(t, os.Rename(path, path+"-previous"))
				require.NoError(t, fsops.SymlinkOrJunction(user, path))
				return failure
			})
			require.ErrorIs(t, err, failure)
			var recovery *restoreError
			require.ErrorAs(t, err, &recovery)
			t.Cleanup(func() { require.NoError(t, os.RemoveAll(recovery.backupDir)) })
			assert.FileExists(t, filepath.Join(recovery.backupDir, "root-1", "old"))
			assert.FileExists(t, filepath.Join(recovery.backupDir, "meta", "manifest-stdx"))
			after, err := os.Stat(userFile)
			require.NoError(t, err)
			assert.True(t, os.SameFile(before, after))
			entries, err := os.ReadDir(user)
			require.NoError(t, err)
			if redirect == "parent" {
				require.Len(t, entries, 2)
				data, err := os.ReadFile(filepath.Join(user, "stdx", "old"))
				require.NoError(t, err)
				assert.Equal(t, "external", string(data))
			} else {
				require.Len(t, entries, 1, "restore must not write backup contents through the link")
			}
		})
	}
}

func TestRemoveRefusesUnreadableSharedOwnershipBeforeMutation(t *testing.T) {
	roots := linkRoots(t)
	owned := filepath.Join(roots.StdxDir, "owned")
	require.NoError(t, os.WriteFile(owned, []byte("old"), 0o644))
	require.NoError(t, WriteManifest(roots.TcDir, Stdx, []string{"owned"}))
	index := metaPath(roots.TcDir, componentsFile)
	require.NoError(t, os.Remove(index))
	require.NoError(t, os.Mkdir(index, 0o755))
	require.Error(t, Remove(roots, Stdx))
	assert.FileExists(t, owned)
	assert.True(t, IsInstalled(roots.TcDir, Stdx))
}

func TestFirstInstallRestoreRefusesNewParentLink(t *testing.T) {
	roots := linkRoots(t)
	roots.StdxDir = filepath.Join(t.TempDir(), "missing", "stdx")
	user := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(user, "stdx"), 0o755))
	userFile := filepath.Join(user, "stdx", "user-owned")
	require.NoError(t, os.WriteFile(userFile, []byte("user"), 0o644))
	failure := errors.New("first install failed after parent redirection")
	err := ApplyChanges(roots, []Name{Stdx}, func() error {
		require.NoError(t, fsops.SymlinkOrJunction(user, filepath.Dir(roots.StdxDir)))
		return failure
	})
	require.ErrorIs(t, err, failure)
	var recovery *restoreError
	require.ErrorAs(t, err, &recovery)
	t.Cleanup(func() { require.NoError(t, os.RemoveAll(recovery.backupDir)) })
	assert.FileExists(t, userFile)
}
