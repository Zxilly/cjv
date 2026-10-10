//go:build !windows

package lifecycle

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const linkedRemovalName = "user-sdk"

type linkedRemovalFixture struct {
	home, source string
	sourceInfo   os.FileInfo
	roots        component.Roots
	sf           *config.SettingsFile
	before       *config.Settings
	beforeBytes  []byte
	update       config.SettingsUpdate
}

func newLinkedRemovalFixture(t *testing.T, kind string) linkedRemovalFixture {
	t.Helper()
	f := linkedRemovalFixture{home: t.TempDir(), source: t.TempDir()}
	config.IsolateForTest(t, f.home)
	require.NoError(t, config.EnsureDirs())
	var err error
	f.roots, err = component.RootsFor(linkedRemovalName)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(f.source, "user-content"), []byte("user SDK\x00bytes"), 0o640))
	f.sourceInfo, err = os.Stat(f.source)
	require.NoError(t, err)
	target := f.source
	switch kind {
	case "relative":
		target, err = filepath.Rel(filepath.Dir(f.roots.TcDir), f.source)
		require.NoError(t, err)
	case "broken":
		target = filepath.Join(f.source, "missing")
	}
	if err := os.Symlink(target, f.roots.TcDir); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	f.sf, err = config.DefaultSettingsFile()
	require.NoError(t, err)
	name := linkedRemovalName
	_, err = f.sf.Update(config.SettingsUpdate{DefaultToolchain: &name, Overrides: map[string]string{
		filepath.Join(f.home, "project"): name, filepath.Join(f.home, "unrelated"): "another-sdk",
	}})
	require.NoError(t, err)
	f.before, err = f.sf.Load()
	require.NoError(t, err)
	f.beforeBytes, err = os.ReadFile(f.sf.Path())
	require.NoError(t, err)
	f.update, err = referencesAfterRemoval(f.before, name)
	require.NoError(t, err)
	return f
}

func (f linkedRemovalFixture) assertSourcePreserved(t *testing.T) {
	t.Helper()
	info, err := os.Stat(f.source)
	require.NoError(t, err)
	assert.True(t, os.SameFile(f.sourceInfo, info))
	assert.Equal(t, f.sourceInfo.Mode(), info.Mode())
	data, err := os.ReadFile(filepath.Join(f.source, "user-content"))
	require.NoError(t, err)
	assert.Equal(t, []byte("user SDK\x00bytes"), data)
}

func (f linkedRemovalFixture) assertSettingsPreserved(t *testing.T) {
	t.Helper()
	data, err := os.ReadFile(f.sf.Path())
	require.NoError(t, err)
	assert.Equal(t, f.beforeBytes, data)
}

func (f linkedRemovalFixture) assertReferencesCleared(t *testing.T) {
	t.Helper()
	settings, err := config.LoadSettings(f.sf.Path())
	require.NoError(t, err)
	assert.Empty(t, settings.DefaultToolchain)
	assert.NotContains(t, settings.Overrides, filepath.Join(f.home, "project"))
	assert.Equal(t, "another-sdk", settings.Overrides[filepath.Join(f.home, "unrelated")])
}

func (f linkedRemovalFixture) assertNoJournal(t *testing.T) {
	t.Helper()
	entries, err := os.ReadDir(filepath.Dir(f.roots.TcDir))
	require.NoError(t, err)
	for _, entry := range entries {
		assert.False(t, strings.HasPrefix(entry.Name(), config.TxTempPrefix), entry.Name())
	}
}

func TestLinkedToolchainRemovalPreservesUserSource(t *testing.T) {
	for _, kind := range []string{"absolute", "relative", "broken"} {
		t.Run(kind, func(t *testing.T) {
			f := newLinkedRemovalFixture(t, kind)
			require.NoError(t, RemoveToolchain(linkedRemovalName))
			_, err := os.Lstat(f.roots.TcDir)
			require.ErrorIs(t, err, os.ErrNotExist)
			f.assertSourcePreserved(t)
			f.assertReferencesCleared(t)
			f.assertNoJournal(t)
			require.NoError(t, toolchain.RecoverHome())
			_, err = os.Lstat(f.roots.TcDir)
			require.ErrorIs(t, err, os.ErrNotExist, "recovery must not recreate an unlinked SDK")
		})
	}
}

func TestLinkedToolchainUnlinkPublishesReferencesFirst(t *testing.T) {
	f := newLinkedRemovalFixture(t, "absolute")
	require.NoError(t, unlinkToolchainLocked(f.roots, f.sf, f.before, f.update, func(root *os.Root, name string) error {
		f.assertReferencesCleared(t)
		f.assertNoJournal(t)
		_, err := root.Readlink(name)
		require.NoError(t, err, "only the final managed link is removed")
		return unlinkToolchainEntry(root, name)
	}))
	f.assertSourcePreserved(t)
}

func TestLinkedToolchainUnlinkFailureRestoresSettings(t *testing.T) {
	f := newLinkedRemovalFixture(t, "absolute")
	before, err := os.Lstat(f.roots.TcDir)
	require.NoError(t, err)
	failure := errors.New("unlink denied")
	err = unlinkToolchainLocked(f.roots, f.sf, f.before, f.update, func(*os.Root, string) error {
		f.assertReferencesCleared(t)
		return failure
	})
	require.ErrorIs(t, err, failure)
	after, err := os.Lstat(f.roots.TcDir)
	require.NoError(t, err)
	assert.True(t, os.SameFile(before, after))
	f.assertSettingsPreserved(t)
	f.assertSourcePreserved(t)
	f.assertNoJournal(t)
}

func TestLinkedToolchainUnlinkErrorAfterRemovalKeepsReferencesCleared(t *testing.T) {
	f := newLinkedRemovalFixture(t, "absolute")
	failure := errors.New("error after unlink")
	err := unlinkToolchainLocked(f.roots, f.sf, f.before, f.update, func(root *os.Root, name string) error {
		require.NoError(t, unlinkToolchainEntry(root, name))
		return failure
	})
	require.ErrorIs(t, err, failure)
	require.ErrorContains(t, err, "settings left cleared")
	f.assertReferencesCleared(t)
	f.assertSourcePreserved(t)
	_, err = os.Lstat(f.roots.TcDir)
	require.ErrorIs(t, err, os.ErrNotExist)
}

type linkedRemovalSettingsFault struct {
	*config.SettingsFile
	update func(config.SettingsUpdate) (bool, error)
	save   func(*config.Settings) error
}

func (sf linkedRemovalSettingsFault) Update(update config.SettingsUpdate) (bool, error) {
	if sf.update != nil {
		return sf.update(update)
	}
	return sf.SettingsFile.Update(update)
}

func (sf linkedRemovalSettingsFault) Save(settings *config.Settings) error {
	if sf.save != nil {
		return sf.save(settings)
	}
	return sf.SettingsFile.Save(settings)
}

func TestLinkedToolchainSettingsFailureKeepsLink(t *testing.T) {
	for _, published := range []bool{false, true} {
		t.Run(map[bool]string{false: "before-publication", true: "after-publication"}[published], func(t *testing.T) {
			f := newLinkedRemovalFixture(t, "absolute")
			failure := errors.New("settings update failed")
			sf := linkedRemovalSettingsFault{SettingsFile: f.sf, update: func(update config.SettingsUpdate) (bool, error) {
				if published {
					_, err := f.sf.Update(update)
					require.NoError(t, err)
				}
				return published, failure
			}}
			err := unlinkToolchainLocked(f.roots, sf, f.before, f.update, func(*os.Root, string) error {
				t.Fatal("must not unlink after a settings error")
				return nil
			})
			require.ErrorIs(t, err, failure)
			_, err = os.Readlink(f.roots.TcDir)
			require.NoError(t, err)
			f.assertSettingsPreserved(t)
			f.assertSourcePreserved(t)
		})
	}
}

func TestLinkedToolchainSettingsRestoreFailureIsReported(t *testing.T) {
	f := newLinkedRemovalFixture(t, "absolute")
	restoreFailure, unlinkFailure := errors.New("settings restore failed"), errors.New("unlink denied")
	sf := linkedRemovalSettingsFault{SettingsFile: f.sf, save: func(*config.Settings) error { return restoreFailure }}
	err := unlinkToolchainLocked(f.roots, sf, f.before, f.update, func(*os.Root, string) error { return unlinkFailure })
	require.ErrorIs(t, err, unlinkFailure)
	require.ErrorIs(t, err, restoreFailure)
	_, err = os.Readlink(f.roots.TcDir)
	require.NoError(t, err)
	f.assertReferencesCleared(t)
	f.assertSourcePreserved(t)
}

func TestLinkedToolchainChangedEntryIsNotUnlinked(t *testing.T) {
	for _, kind := range []string{"directory", "file", "symlink", "missing"} {
		t.Run(kind, func(t *testing.T) {
			f := newLinkedRemovalFixture(t, "absolute")
			sf := linkedRemovalSettingsFault{SettingsFile: f.sf, update: func(update config.SettingsUpdate) (bool, error) {
				changed, err := f.sf.Update(update)
				require.NoError(t, err)
				require.NoError(t, os.Remove(f.roots.TcDir))
				switch kind {
				case "directory":
					require.NoError(t, os.Mkdir(f.roots.TcDir, 0o755))
				case "file":
					require.NoError(t, os.WriteFile(f.roots.TcDir, []byte("replacement"), 0o644))
				case "symlink":
					require.NoError(t, os.Symlink("different-target", f.roots.TcDir))
				}
				return changed, nil
			}}
			err := unlinkToolchainLocked(f.roots, sf, f.before, f.update, func(*os.Root, string) error {
				t.Fatal("changed entry must not be removed")
				return nil
			})
			require.Error(t, err)
			f.assertReferencesCleared(t)
			f.assertSourcePreserved(t)
			switch kind {
			case "directory":
				assert.DirExists(t, f.roots.TcDir)
			case "file":
				data, err := os.ReadFile(f.roots.TcDir)
				require.NoError(t, err)
				assert.Equal(t, "replacement", string(data))
			case "symlink":
				target, err := os.Readlink(f.roots.TcDir)
				require.NoError(t, err)
				assert.Equal(t, "different-target", target)
			}
		})
	}
}

func TestLinkedToolchainMovedParentKeepsReferencesCleared(t *testing.T) {
	for _, timing := range []string{"after-settings", "unlink-error"} {
		for _, replacement := range []bool{false, true} {
			t.Run(timing+map[bool]string{false: "/missing", true: "/replacement"}[replacement], func(t *testing.T) {
				f := newLinkedRemovalFixture(t, "absolute")
				parent := filepath.Dir(f.roots.TcDir)
				moved := parent + "-moved"
				moveParent := func() {
					require.NoError(t, os.Rename(parent, moved))
					if replacement {
						require.NoError(t, os.Mkdir(parent, 0o755))
						require.NoError(t, os.Symlink("other-source", f.roots.TcDir))
					}
				}
				sf := linkedRemovalSettingsFault{SettingsFile: f.sf, update: func(update config.SettingsUpdate) (bool, error) {
					changed, err := f.sf.Update(update)
					require.NoError(t, err)
					if timing == "after-settings" {
						moveParent()
					}
					return changed, nil
				}}
				failure := errors.New("unlink failed after parent moved")
				err := unlinkToolchainLocked(f.roots, sf, f.before, f.update, func(*os.Root, string) error {
					require.Equal(t, "unlink-error", timing, "must not unlink a detached original entry")
					moveParent()
					return failure
				})
				require.ErrorContains(t, err, "settings left cleared")
				if timing == "unlink-error" {
					require.ErrorIs(t, err, failure)
				}
				target, err := os.Readlink(filepath.Join(moved, linkedRemovalName))
				require.NoError(t, err)
				assert.Equal(t, f.source, target, "the detached original link must remain untouched")
				if replacement {
					target, err := os.Readlink(f.roots.TcDir)
					require.NoError(t, err)
					assert.Equal(t, "other-source", target)
				}
				f.assertReferencesCleared(t)
				f.assertSourcePreserved(t)
			})
		}
	}
}

func TestLinkedToolchainRemovalRejectsComponentRoots(t *testing.T) {
	for _, subdir := range []string{config.DocsSubdir, config.StdxSubdir} {
		for _, kind := range []string{"directory", "symlink", "broken-symlink"} {
			t.Run(subdir+"/"+kind, func(t *testing.T) {
				f := newLinkedRemovalFixture(t, "absolute")
				path := filepath.Join(f.home, subdir, linkedRemovalName)
				require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
				if kind == "directory" {
					require.NoError(t, os.Mkdir(path, 0o755))
				} else {
					target := f.source
					if kind == "broken-symlink" {
						target += "/missing"
					}
					require.NoError(t, os.Symlink(target, path))
				}
				before, err := os.Lstat(path)
				require.NoError(t, err)
				require.ErrorContains(t, RemoveToolchain(linkedRemovalName), "component root")
				after, err := os.Lstat(path)
				require.NoError(t, err)
				assert.True(t, os.SameFile(before, after))
				_, err = os.Readlink(f.roots.TcDir)
				require.NoError(t, err)
				f.assertSettingsPreserved(t)
				f.assertSourcePreserved(t)
				f.assertNoJournal(t)
			})
		}
	}
}

func TestLinkedToolchainRemovalRejectsLinkedParents(t *testing.T) {
	for _, subdir := range []string{config.ToolchainsSubdir, config.DocsSubdir, config.StdxSubdir} {
		t.Run(subdir, func(t *testing.T) {
			f := newLinkedRemovalFixture(t, "absolute")
			parent, moved := filepath.Join(f.home, subdir), filepath.Join(f.home, subdir+"-original")
			require.NoError(t, os.MkdirAll(parent, 0o755))
			require.NoError(t, os.Rename(parent, moved))
			require.NoError(t, os.Symlink(moved, parent))
			err := unlinkToolchainLocked(f.roots, f.sf, f.before, f.update, unlinkToolchainEntry)
			require.ErrorContains(t, err, "linked parent")
			f.assertSettingsPreserved(t)
			f.assertSourcePreserved(t)
			_, err = os.Readlink(f.roots.TcDir)
			require.NoError(t, err)
		})
	}
}

func TestLinkedToolchainUnlinkIsFileOnlyAndScoped(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	require.NoError(t, err)
	defer root.Close()
	require.NoError(t, root.Mkdir("replacement-directory", 0o755))
	require.Error(t, unlinkToolchainEntry(root, "replacement-directory"))
	assert.DirExists(t, filepath.Join(dir, "replacement-directory"))
	for _, name := range []string{".", "..", "../outside", "parent/entry", "/absolute"} {
		require.ErrorIs(t, unlinkToolchainEntry(root, name), os.ErrInvalid)
	}
	info, err := root.Lstat("replacement-directory")
	require.NoError(t, err)
	assert.Nil(t, linkedToolchainRemover(info), "owned SDK directories retain their transaction")
}

func TestLinkedToolchainUnlinkCrashLeavesReferencesCleared(t *testing.T) {
	if home := os.Getenv("CJV_TEST_UNLINK_HOME"); home != "" {
		config.IsolateForTest(t, home)
		roots, err := component.RootsFor(linkedRemovalName)
		require.NoError(t, err)
		sf, before, err := config.LoadDefaultSettings()
		require.NoError(t, err)
		update, err := referencesAfterRemoval(before, linkedRemovalName)
		require.NoError(t, err)
		_ = unlinkToolchainLocked(roots, sf, before, update, func(root *os.Root, name string) error {
			if os.Getenv("CJV_TEST_UNLINK_STAGE") == "after-unlink" {
				require.NoError(t, unlinkToolchainEntry(root, name))
			}
			os.Exit(42)
			return nil
		})
		t.Fatal("expected process interruption")
	}
	for _, stage := range []string{"before-unlink", "after-unlink"} {
		t.Run(stage, func(t *testing.T) {
			f := newLinkedRemovalFixture(t, "absolute")
			cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestLinkedToolchainUnlinkCrashLeavesReferencesCleared$")
			cmd.Env = append(os.Environ(), "CJV_TEST_UNLINK_HOME="+f.home, "CJV_TEST_UNLINK_STAGE="+stage)
			output, err := cmd.CombinedOutput()
			var exitErr *exec.ExitError
			require.ErrorAs(t, err, &exitErr, "%s", output)
			require.Equal(t, 42, exitErr.ExitCode(), "%s", output)
			require.NoError(t, toolchain.RecoverHome())
			_, err = os.Lstat(f.roots.TcDir)
			if stage == "after-unlink" {
				require.ErrorIs(t, err, os.ErrNotExist)
			} else {
				require.NoError(t, err)
			}
			f.assertReferencesCleared(t)
			f.assertSourcePreserved(t)
			f.assertNoJournal(t)
		})
	}
}
