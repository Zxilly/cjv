package fsops

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MoveTree stages extracted archives into install roots: the tree is merged
// entry by entry, existing entries are replaced, and the placed files are
// reported for the component manifest.

func TestMoveTreeMovesNestedTreeAndRecordsFiles(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "sdk")
	deep := filepath.Join(src, "lib", "cangjie", "runtime")
	require.NoError(t, os.MkdirAll(deep, 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(src, "bin"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(src, "empty"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "bin", "cjc"), []byte("binary"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(deep, "leaf.so"), []byte("deep"), 0o644))

	paths, err := MoveTree(src, dst)
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{"bin/cjc", "lib/cangjie/runtime/leaf.so"}, paths, "directories are not recorded")
	assertFileContent(t, filepath.Join(dst, "bin", "cjc"), "binary")
	assertFileContent(t, filepath.Join(dst, "lib", "cangjie", "runtime", "leaf.so"), "deep")
	assert.DirExists(t, filepath.Join(dst, "empty"))
	assert.NoFileExists(t, filepath.Join(src, "bin", "cjc"), "a move leaves no file behind")
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(dst, "bin", "cjc"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())
	}
}

func TestMoveTreeOverwritesExistingEntry(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(src, "bin"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(dst, "bin"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "bin", "cjc"), []byte("fresh"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dst, "bin", "cjc"), []byte("stale"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dst, "bin", "keep"), []byte("untouched"), 0o644))

	paths, err := MoveTree(src, dst)

	require.NoError(t, err)
	assert.Equal(t, []string{"bin/cjc"}, paths)
	assertFileContent(t, filepath.Join(dst, "bin", "cjc"), "fresh")
	assertFileContent(t, filepath.Join(dst, "bin", "keep"), "untouched")
}

func TestMoveTreeReplacesDirectoryWithFile(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "lib"), []byte("file"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dst, "lib", "old"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dst, "lib", "old", "x"), []byte("x"), 0o644))

	_, err := MoveTree(src, dst)

	require.NoError(t, err)
	assertFileContent(t, filepath.Join(dst, "lib"), "file")
}

func TestMoveTreeRequiresSourceDirectoryAndWritableDestination(t *testing.T) {
	_, err := MoveTree(filepath.Join(t.TempDir(), "missing"), t.TempDir())
	require.Error(t, err)

	file := filepath.Join(t.TempDir(), "file")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))
	_, err = MoveTree(file, t.TempDir())
	require.Error(t, err)

	parentFile := filepath.Join(t.TempDir(), "not-a-directory")
	require.NoError(t, os.WriteFile(parentFile, []byte("file"), 0o644))
	_, err = MoveTree(t.TempDir(), filepath.Join(parentFile, "dest"))
	require.Error(t, err)
}

// SDK archives ship internal cross-directory relative symlinks such as
// third_party/llvm/lib/foo.so -> ../../../runtime/lib/foo.so. These resolve
// inside the archive root and must be allowed.
func TestMoveTreeKeepsInternalRelativeSymlink(t *testing.T) {
	src := t.TempDir()
	deepDir := filepath.Join(src, "third_party", "llvm", "lib")
	runtimeDir := filepath.Join(src, "runtime", "lib")
	require.NoError(t, os.MkdirAll(deepDir, 0o755))
	require.NoError(t, os.MkdirAll(runtimeDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(runtimeDir, "libfoo.so"), []byte("x"), 0o644))
	relTarget := filepath.Join("..", "..", "..", "runtime", "lib", "libfoo.so")
	if err := os.Symlink(relTarget, filepath.Join(deepDir, "libfoo.so")); err != nil {
		t.Skipf("symlink creation requires privileges on this platform: %v", err)
	}

	dst := filepath.Join(t.TempDir(), "sdk")
	paths, err := MoveTree(src, dst)
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{"runtime/lib/libfoo.so", "third_party/llvm/lib/libfoo.so"}, paths)
	gotTarget, err := os.Readlink(filepath.Join(dst, "third_party", "llvm", "lib", "libfoo.so"))
	require.NoError(t, err)
	assert.Equal(t, filepath.ToSlash(relTarget), filepath.ToSlash(gotTarget))
	assertFileContent(t, filepath.Join(dst, "third_party", "llvm", "lib", "libfoo.so"), "x")
}

func TestMoveTreeRejectsUnsafeSymlinkTargets(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	absoluteLink := filepath.Join(src, "absolute-link")
	if err := os.Symlink(filepath.Join(src, "target.txt"), absoluteLink); err != nil {
		t.Skipf("symlink creation requires privileges on this platform: %v", err)
	}

	paths, err := MoveTree(src, dst)
	require.Error(t, err)
	assert.Empty(t, paths)
	assert.NoFileExists(t, filepath.Join(dst, "absolute-link"))

	src = t.TempDir()
	dst = t.TempDir()
	require.NoError(t, os.Symlink(filepath.Join("..", "target.txt"), filepath.Join(src, "parent-link")))

	paths, err = MoveTree(src, dst)
	require.Error(t, err)
	assert.Empty(t, paths)
	assert.NoFileExists(t, filepath.Join(dst, "parent-link"))
}

func TestMoveTreeRejectsUnsafeSymlinkWithoutDeletingExistingDestination(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	existingPath := filepath.Join(dst, "lib")
	require.NoError(t, os.WriteFile(existingPath, []byte("old"), 0o644))
	if err := os.Symlink(filepath.Join(src, "target.txt"), filepath.Join(src, "lib")); err != nil {
		t.Skipf("symlink creation requires privileges on this platform: %v", err)
	}

	paths, err := MoveTree(src, dst)
	require.Error(t, err)
	assert.Empty(t, paths)
	assertFileContent(t, existingPath, "old")
}

// A rename across filesystems fails with EXDEV; MoveTree then copies the
// entry instead. tmpfs at /dev/shm is a different filesystem from the test
// temp directory on most Linux hosts; elsewhere the fallback is skipped.
func TestMoveTreeCopiesAcrossFilesystems(t *testing.T) {
	other := crossDeviceDir(t)

	src := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(src, "bin"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "bin", "cjc"), []byte("binary"), 0o755))
	if err := os.Symlink("cjc", filepath.Join(src, "bin", "cjc-link")); err != nil {
		t.Skipf("symlink creation requires privileges on this platform: %v", err)
	}

	dst := filepath.Join(other, "sdk")
	paths, err := MoveTree(src, dst)
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{"bin/cjc", "bin/cjc-link"}, paths)
	assertFileContent(t, filepath.Join(dst, "bin", "cjc"), "binary")
	gotTarget, err := os.Readlink(filepath.Join(dst, "bin", "cjc-link"))
	require.NoError(t, err)
	assert.Equal(t, "cjc", gotTarget)
	info, err := os.Stat(filepath.Join(dst, "bin", "cjc"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())
}

// CopyTree backs up and restores component roots, so what it produces must
// be what was there: modes, symlinks with their original targets, nesting.

func TestCopyTreeCopiesSingleFileIntoMissingParent(t *testing.T) {
	src := filepath.Join(t.TempDir(), "source.txt")
	dst := filepath.Join(t.TempDir(), "nested", "copy.txt")
	require.NoError(t, os.WriteFile(src, []byte("content"), 0o644))

	require.NoError(t, CopyTree(src, dst))

	assertFileContent(t, dst, "content")
	assert.FileExists(t, src, "a copy leaves the source in place")
}

func TestCopyTreeCopiesDirectoryTreePreservingModes(t *testing.T) {
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "dst")
	require.NoError(t, os.MkdirAll(filepath.Join(src, "nested", "deeper"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(src, "empty"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "nested", "file.txt"), []byte("tree"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "nested", "deeper", "tool"), []byte("bin"), 0o755))

	require.NoError(t, CopyTree(src, dst))

	assertFileContent(t, filepath.Join(dst, "nested", "file.txt"), "tree")
	assertFileContent(t, filepath.Join(dst, "nested", "deeper", "tool"), "bin")
	assert.DirExists(t, filepath.Join(dst, "empty"))
	if runtime.GOOS != "windows" {
		info, err := os.Stat(filepath.Join(dst, "nested", "deeper", "tool"))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o755), info.Mode().Perm())
	}
}

func TestCopyTreePreservesReadOnlyDirectoryAfterFillingIt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory modes are not enforced on Windows")
	}
	if os.Getuid() == 0 {
		t.Skip("root bypasses directory permissions")
	}
	src := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(src, "ro"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "ro", "file"), []byte("x"), 0o644))
	require.NoError(t, os.Chmod(filepath.Join(src, "ro"), 0o555))
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(src, "ro"), 0o755) })

	dst := filepath.Join(t.TempDir(), "dst")
	require.NoError(t, CopyTree(src, dst))
	t.Cleanup(func() { _ = os.Chmod(filepath.Join(dst, "ro"), 0o755) })

	assertFileContent(t, filepath.Join(dst, "ro", "file"), "x")
	info, err := os.Stat(filepath.Join(dst, "ro"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o555), info.Mode().Perm())
}

func TestCopyTreeReplacesExistingFilesAndMergesDirectories(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(src, "dynamic"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "dynamic", "libfoo.so"), []byte("backup"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dst, "dynamic"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dst, "dynamic", "libfoo.so"), []byte("live"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dst, "dynamic", "other.so"), []byte("other"), 0o644))

	require.NoError(t, CopyTree(src, dst))

	assertFileContent(t, filepath.Join(dst, "dynamic", "libfoo.so"), "backup")
	assertFileContent(t, filepath.Join(dst, "dynamic", "other.so"), "other")
}

func TestCopyTreeReproducesSymlinksVerbatim(t *testing.T) {
	srcDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(srcDir, "target.txt"), []byte("target"), 0o644))
	if err := os.Symlink("target.txt", filepath.Join(srcDir, "link.txt")); err != nil {
		t.Skipf("symlink creation requires privileges on this platform: %v", err)
	}
	// Backups must keep links that point outside the tree (a locally linked
	// component root is such a link), which MoveTree would reject.
	outside := t.TempDir()
	require.NoError(t, os.Symlink(outside, filepath.Join(srcDir, "linked-root")))

	dst := filepath.Join(t.TempDir(), "copy")
	require.NoError(t, CopyTree(srcDir, dst))

	gotTarget, err := os.Readlink(filepath.Join(dst, "link.txt"))
	require.NoError(t, err)
	assert.Equal(t, "target.txt", gotTarget)
	gotOutside, err := os.Readlink(filepath.Join(dst, "linked-root"))
	require.NoError(t, err)
	assert.Equal(t, outside, gotOutside)

	// A single symlink is copied as a link too, replacing what is at dst.
	linkCopy := filepath.Join(t.TempDir(), "link-copy.txt")
	require.NoError(t, os.WriteFile(linkCopy, []byte("stale"), 0o644))
	require.NoError(t, CopyTree(filepath.Join(srcDir, "link.txt"), linkCopy))
	gotTarget, err = os.Readlink(linkCopy)
	require.NoError(t, err)
	assert.Equal(t, "target.txt", gotTarget)
}

func TestCopyTreeReportsMissingSource(t *testing.T) {
	err := CopyTree(filepath.Join(t.TempDir(), "missing"), filepath.Join(t.TempDir(), "copy"))
	require.Error(t, err)
}

func assertFileContent(t *testing.T, path, expected string) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "file should exist: %s", path)
	assert.Equal(t, expected, string(data))
}
