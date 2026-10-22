//go:build unix

package lifecycle

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func installHistoryReadOnlyTree(t *testing.T) *historyUpdateFixture {
	t.Helper()
	f := newHistoryUpdateFixture(t)
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts", Targets: []string{"ohos"}}, Options{}))
	dir := filepath.Join(f.home, "toolchains", "sts", "readonly")
	require.NoError(t, os.Mkdir(dir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "payload"), []byte("retained"), 0o444))
	require.NoError(t, os.Chmod(dir, 0o555))
	t.Cleanup(func() {
		_ = filepath.WalkDir(f.home, func(path string, entry fs.DirEntry, err error) error {
			if err == nil && entry.IsDir() {
				_ = os.Chmod(path, 0o700)
			}
			return nil
		})
	})
	f.republishTarget.Store(true)
	return f
}

func requireHistoryCleanupComplete(t *testing.T, home string) {
	t.Helper()
	require.NoError(t, toolchain.RecoverHome())
	require.NoError(t, filepath.WalkDir(home, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		require.False(t, strings.HasSuffix(entry.Name(), config.StagingSuffix), "staging residue: %s", path)
		require.False(t, strings.HasPrefix(entry.Name(), config.TxTempPrefix), "transaction residue: %s", path)
		require.False(t, strings.HasPrefix(entry.Name(), ".cjv-rollback-"), "rollback scratch: %s", path)
		return nil
	}))
}

func TestHistoryPrunesReadOnlySnapshot(t *testing.T) {
	f := installHistoryReadOnlyTree(t)
	_, err := UpdateInstalled(t.Context(), parse(t, "sts"), Options{})
	require.NoError(t, err)
	removed, err := PruneHistory(t.Context(), "sts", 0)
	require.NoError(t, err)
	require.Equal(t, 1, removed)
	entries, err := ListHistory(t.Context(), "sts")
	require.NoError(t, err)
	require.Empty(t, entries)
	requireHistoryCleanupComplete(t, f.home)
}

func TestHistoryRollbackCleansReadOnlyBackup(t *testing.T) {
	f := installHistoryReadOnlyTree(t)
	_, err := UpdateInstalled(t.Context(), parse(t, "sts"), Options{})
	require.NoError(t, err)
	_, err = RollbackHistory(t.Context(), "sts", "", Options{})
	require.NoError(t, err)
	requireHistoryCleanupComplete(t, f.home)
}

func TestHistoryFailedPublicationCleansReadOnlyTrees(t *testing.T) {
	for _, phase := range []string{"staging", "finalize"} {
		t.Run(phase, func(t *testing.T) {
			f := installHistoryReadOnlyTree(t)
			_, err := UpdateInstalled(t.Context(), parse(t, "sts"), Options{})
			require.NoError(t, err)
			entries, err := ListHistory(t.Context(), "sts")
			require.NoError(t, err)
			require.Len(t, entries, 1)
			before := readRelease(t, f.home, "sts-"+f.host+"-ohos")
			priorStaging, priorFinalize := afterStagingHook, afterFinalizeHook
			t.Cleanup(func() { afterStagingHook, afterFinalizeHook = priorStaging, priorFinalize })
			injected := errors.New("stop history publication")
			if phase == "staging" {
				afterStagingHook = func() error { return injected }
			} else {
				afterFinalizeHook = func() error { return injected }
			}
			_, err = RollbackHistory(t.Context(), "sts", entries[0].ID, Options{})
			require.ErrorIs(t, err, injected)
			afterStagingHook, afterFinalizeHook = priorStaging, priorFinalize
			requireHistoryCleanupComplete(t, f.home)
			require.Equal(t, before, readRelease(t, f.home, "sts-"+f.host+"-ohos"))
			after, err := ListHistory(t.Context(), "sts")
			require.NoError(t, err)
			require.Equal(t, entries, after)
		})
	}
}

func TestHistoryRollbackFailureCleansReadOnlyScratch(t *testing.T) {
	f := installHistoryReadOnlyTree(t)
	_, err := UpdateInstalled(t.Context(), parse(t, "sts"), Options{})
	require.NoError(t, err)
	// Current metadata is read after the snapshot has been copied to scratch.
	// Failing here must remove the preserved read-only scratch directories.
	record := filepath.Join(f.home, "toolchains", "sts", ".cjv", "toolchain.toml")
	require.NoError(t, os.WriteFile(record, []byte("invalid = ["), 0o644))
	_, err = RollbackHistory(t.Context(), "sts", "", Options{})
	require.Error(t, err)
	requireHistoryCleanupComplete(t, f.home)
}

func TestHistoryRecoveryCleansReadOnlyStaging(t *testing.T) {
	f := installHistoryReadOnlyTree(t)
	paths := []string{
		filepath.Join("history", "sts", "20260101T000000.000000000Z-0123456789ab.staging"),
		filepath.Join("toolchains", "sts.staging"),
		filepath.Join("docs", "sts.staging"),
		filepath.Join("stdx", "sts.staging"),
	}
	for _, path := range paths {
		require.NoError(t, copyHistoryTree(t.Context(), filepath.Join(f.home, "toolchains", "sts"), filepath.Join(f.home, path), nil))
	}
	requireHistoryCleanupComplete(t, f.home)
	info, err := os.Stat(filepath.Join(f.home, "toolchains", "sts", "readonly"))
	require.NoError(t, err)
	require.Equal(t, os.FileMode(0o555), info.Mode().Perm(), "cleanup must not change the installed tree")
}

func TestHistoryRollbackPreservesPermissionsAcrossUmask(t *testing.T) {
	oldMask := unix.Umask(0o022)
	t.Cleanup(func() { unix.Umask(oldMask) })
	home := updateHome(t)
	selectRelease(t, "1.0.0")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, Options{}))
	compiler := filepath.Join(home, "toolchains", "sts", "bin", "cjc")
	require.NoError(t, os.Chmod(compiler, 0o755))
	require.NoError(t, os.Chmod(filepath.Dir(compiler), 0o751))
	selectRelease(t, "2.0.0")
	_, err := UpdateInstalled(t.Context(), parse(t, "sts"), Options{})
	require.NoError(t, err)
	entries, err := ListHistory(t.Context(), "sts")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	unix.Umask(0o077)
	_, err = RollbackHistory(t.Context(), "sts", entries[0].ID, Options{})
	require.NoError(t, err, "the current shell umask must not change an unchanged snapshot's checksum")
	for path, expected := range map[string]os.FileMode{compiler: 0o755, filepath.Dir(compiler): 0o751} {
		info, err := os.Stat(path)
		require.NoError(t, err)
		require.Equal(t, expected, info.Mode().Perm())
	}
}

func TestCopyHistoryTreePreservesReadOnlyDirectories(t *testing.T) {
	oldMask := unix.Umask(0o077)
	t.Cleanup(func() { unix.Umask(oldMask) })
	base := t.TempDir()
	source, dest := filepath.Join(base, "source"), filepath.Join(base, "copy")
	child := filepath.Join("read-only", "nested")
	require.NoError(t, os.MkdirAll(filepath.Join(source, child), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(source, child, "data"), []byte("retained bytes"), 0o600))
	// Make cleanup possible even when assertions fail after copying read-only directories.
	t.Cleanup(func() {
		for _, root := range []string{source, dest} {
			for _, dir := range []string{root, filepath.Join(root, "read-only"), filepath.Join(root, child)} {
				_ = os.Chmod(dir, 0o700)
			}
		}
	})
	require.NoError(t, os.Chmod(filepath.Join(source, child, "data"), 0o444))
	for _, dir := range []string{filepath.Join(source, child), filepath.Join(source, "read-only"), source} {
		require.NoError(t, os.Chmod(dir, 0o555))
	}
	require.NoError(t, copyHistoryTree(t.Context(), source, dest, nil))
	before, err := historyTreeDigest(t.Context(), source)
	require.NoError(t, err)
	after, err := historyTreeDigest(t.Context(), dest)
	require.NoError(t, err)
	require.Equal(t, before, after, "copies must retain file bytes and modes, including read-only parents")
}
