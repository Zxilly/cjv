package component

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplacementRestoresExistingComponentWhenIndexUpdateFails(t *testing.T) {
	for _, operation := range []string{"link", "archive"} {
		t.Run(operation, func(t *testing.T) {
			roots := linkRoots(t)
			oldPaths := []string{"dynamic/old.so", "static/old.a"}
			for _, rel := range oldPaths {
				path := filepath.Join(roots.StdxDir, filepath.FromSlash(rel))
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				require.NoError(t, os.WriteFile(path, []byte("old library"), 0o644))
			}
			require.NoError(t, WriteManifest(roots.TcDir, Stdx, oldPaths))

			// A damaged index makes removal fail after the old files and
			// manifest have already been removed. Both replacement operations
			// must restore the installation as it was before the attempt.
			index := metaPath(roots.TcDir, componentsFile)
			require.NoError(t, os.Remove(index))
			require.NoError(t, os.Mkdir(index, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(index, "keep"), []byte("index content"), 0o644))

			var err error
			if operation == "link" {
				source := makeStdxSource(t, "new library")
				_, err = Link(roots, Stdx, source, true)
				assert.FileExists(t, filepath.Join(source, "dynamic", "new library"))
				assert.FileExists(t, filepath.Join(source, "static", "new library"))
			} else {
				archive := buildZip(t, t.TempDir(), "stdx.zip", map[string]string{
					"stdx/dynamic/new.so": "new library",
					"stdx/static/new.a":   "new library",
				})
				err = InstallFromArchive(context.Background(), roots, Stdx, archive, true)
				assert.FileExists(t, archive)
			}
			require.Error(t, err)
			for _, rel := range oldPaths {
				data, readErr := os.ReadFile(filepath.Join(roots.StdxDir, filepath.FromSlash(rel)))
				require.NoError(t, readErr)
				assert.Equal(t, "old library", string(data))
			}
			manifest, readErr := ReadManifest(roots.TcDir, Stdx)
			require.NoError(t, readErr)
			assert.Equal(t, oldPaths, manifest)
			assert.FileExists(t, filepath.Join(index, "keep"))
		})
	}
}

func TestArchiveInstallationFailureRestoresOverwrittenFiles(t *testing.T) {
	for _, installed := range []bool{false, true} {
		name := "first install"
		if installed {
			name = "force reinstall"
		}
		t.Run(name, func(t *testing.T) {
			roots := linkRoots(t)
			docsRoot := filepath.Join(roots.DocsDir, "main")
			require.NoError(t, os.MkdirAll(docsRoot, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(docsRoot, "a.txt"), []byte("original untracked file"), 0o644))
			// Extraction succeeds, but promotion of z/page.txt fails because z
			// is a file. Alphabetical promotion has already overwritten a.txt.
			require.NoError(t, os.WriteFile(filepath.Join(docsRoot, "z"), []byte("original obstruction"), 0o644))
			if installed {
				require.NoError(t, os.WriteFile(filepath.Join(docsRoot, "old.txt"), []byte("old docs"), 0o644))
				require.NoError(t, WriteManifest(roots.TcDir, Docs, []string{"old.txt"}))
			}
			archive := buildZip(t, t.TempDir(), "docs.zip", map[string]string{
				"a.txt":      "new docs",
				"z/page.txt": "new docs",
			})

			err := InstallFromArchive(context.Background(), roots, Docs, archive, installed)
			require.Error(t, err)
			data, readErr := os.ReadFile(filepath.Join(docsRoot, "a.txt"))
			require.NoError(t, readErr)
			assert.Equal(t, "original untracked file", string(data))
			data, readErr = os.ReadFile(filepath.Join(docsRoot, "z"))
			require.NoError(t, readErr)
			assert.Equal(t, "original obstruction", string(data))
			assert.Equal(t, installed, IsInstalled(roots.TcDir, Docs))
			if installed {
				data, readErr = os.ReadFile(filepath.Join(docsRoot, "old.txt"))
				require.NoError(t, readErr)
				assert.Equal(t, "old docs", string(data))
				manifest, readErr := ReadManifest(roots.TcDir, Docs)
				require.NoError(t, readErr)
				assert.Equal(t, []string{"old.txt"}, manifest)
			}
		})
	}
}

func TestForcedLinkFailureRestoresUserOwnedLinks(t *testing.T) {
	oldSource := makeStdxSource(t, "old library")
	newSource := makeStdxSource(t, "new library")
	roots := linkRoots(t)
	linkOK(t, roots, Stdx, oldSource, false)
	index := metaPath(roots.TcDir, componentsFile)
	require.NoError(t, os.Remove(index))
	require.NoError(t, os.Mkdir(index, 0o755))

	_, err := Link(roots, Stdx, newSource, true)
	require.Error(t, err)
	for _, child := range []string{"dynamic", "static"} {
		assertSymlinkTo(t, filepath.Join(roots.StdxDir, child), filepath.Join(oldSource, child))
		assert.FileExists(t, filepath.Join(oldSource, child, "old library"))
		assert.FileExists(t, filepath.Join(newSource, child, "new library"))
	}
	manifest, err := ReadManifest(roots.TcDir, Stdx)
	require.NoError(t, err)
	assert.Equal(t, []string{"dynamic", "static"}, manifest)
}

func TestComponentChangesRetainBackupWhenRestoreFails(t *testing.T) {
	parent := filepath.Join(t.TempDir(), "components")
	roots := Roots{TcDir: t.TempDir(), StdxDir: filepath.Join(parent, "stdx"), DocsDir: t.TempDir()}
	oldPath := filepath.Join(roots.StdxDir, "dynamic", "old.so")
	require.NoError(t, os.MkdirAll(filepath.Dir(oldPath), 0o755))
	require.NoError(t, os.WriteFile(oldPath, []byte("old library"), 0o644))
	require.NoError(t, WriteManifest(roots.TcDir, Stdx, []string{"dynamic/old.so"}))
	archive := buildZip(t, t.TempDir(), "stdx.zip", map[string]string{
		"stdx/dynamic/new.so": "new library",
		"stdx/static/new.a":   "new library",
	})

	var installErr error
	err := ApplyChanges(roots, []Name{Stdx}, func() error {
		if err := InstallFromArchive(context.Background(), roots, Stdx, archive, true); err != nil {
			return err
		}
		// An external filesystem change prevents both the next installation
		// and restoration. The reported backup must still contain the original.
		require.NoError(t, os.RemoveAll(parent))
		require.NoError(t, os.WriteFile(parent, []byte("blocked parent"), 0o644))
		installErr = InstallFromArchive(context.Background(), roots, Stdx, archive, true)
		return installErr
	})

	require.Error(t, installErr)
	require.ErrorIs(t, err, installErr)
	var restoreErr *restoreError
	require.ErrorAs(t, err, &restoreErr)
	t.Cleanup(func() { require.NoError(t, os.RemoveAll(restoreErr.backupDir)) })
	assert.Contains(t, err.Error(), restoreErr.backupDir)
	data, readErr := os.ReadFile(filepath.Join(restoreErr.backupDir, "root-1", "dynamic", "old.so"))
	require.NoError(t, readErr)
	assert.Equal(t, "old library", string(data))
	manifest, readErr := os.ReadFile(filepath.Join(restoreErr.backupDir, "meta", manifestPrefix+string(Stdx)))
	require.NoError(t, readErr)
	assert.Equal(t, "dynamic/old.so\n", string(manifest))
}
