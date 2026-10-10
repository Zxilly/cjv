package fsops

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateLink(t *testing.T) {
	tmp := t.TempDir()
	src := filepath.Join(tmp, "source")
	require.NoError(t, os.WriteFile(src, []byte("binary"), 0o755))

	dst := filepath.Join(tmp, "link")
	require.NoError(t, CreateLink(src, dst))
	assert.FileExists(t, dst)
}

func TestCreateLinkRemovesExistingDestinationBeforeLinkAttempts(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source")
	dst := filepath.Join(dir, "tool.exe")
	require.NoError(t, os.WriteFile(src, []byte("new"), 0o755))
	require.NoError(t, os.WriteFile(dst, []byte("old"), 0o755))

	var order []string
	failing := func(msg string) linkStrategy {
		return func(from, to string) error {
			assert.Equal(t, src, from)
			assert.Equal(t, dst, to, "links must be attempted at the final path")
			_, err := os.Lstat(to)
			require.ErrorIs(t, err, os.ErrNotExist, "old proxy must be removed before each attempt")
			order = append(order, msg)
			return errors.New(msg)
		}
	}
	strategies := []linkStrategy{
		failing("symlink disabled"),
		failing("hard link disabled"),
	}

	err := createLinkWith(strategies, src, dst)
	require.EqualError(t, err, "hard link disabled", "the final link error is reported without copying")
	assert.Equal(t, []string{"symlink disabled", "hard link disabled"}, order)
	_, err = os.Lstat(dst)
	require.ErrorIs(t, err, os.ErrNotExist, "failure does not restore the old proxy")
	data, readErr := os.ReadFile(src)
	require.NoError(t, readErr)
	assert.Equal(t, []byte("new"), data)
	requireProxyEntries(t, dir, "source")
}

func TestCreateLinkFallsBackToHardLink(t *testing.T) {
	tmp := t.TempDir()
	if !hardlinksAvailable(t, tmp) {
		t.Skip("filesystem policy denies hard links")
	}
	src := filepath.Join(tmp, "source")
	dst := filepath.Join(tmp, "link")
	require.NoError(t, os.WriteFile(src, []byte("binary"), 0o755))

	var order []string
	record := func(name string, next linkStrategy) linkStrategy {
		return func(src, dst string) error {
			order = append(order, name)
			return next(src, dst)
		}
	}
	failing := func(string, string) error { return errors.New("unavailable") }

	require.NoError(t, createLinkWith([]linkStrategy{
		record("symlink", failing),
		record("hardlink", hardLinkStrategy),
		record("never", failing),
	}, src, dst))

	assert.Equal(t, []string{"symlink", "hardlink"}, order)
	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, []byte("binary"), data)
	sourceInfo, err := os.Stat(src)
	require.NoError(t, err)
	destinationInfo, err := os.Lstat(dst)
	require.NoError(t, err)
	assert.True(t, os.SameFile(sourceInfo, destinationInfo), "fallback must link, not copy")
	assert.Equal(t, sourceInfo.Mode(), destinationInfo.Mode())
	requireProxyEntries(t, tmp, "source", "link")
}

func TestCreateLinkStopsAfterFirstSuccess(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source")
	dst := filepath.Join(dir, "nested", "proxy")
	require.NoError(t, os.WriteFile(src, []byte("binary"), 0o755))
	require.NoError(t, createLinkWith([]linkStrategy{
		func(from, to string) error {
			assert.Equal(t, dst, to, "no temporary replacement path is used")
			// This test covers strategy ordering; hard-link behavior is tested separately.
			return os.WriteFile(to, []byte("binary"), 0o755)
		},
		func(string, string) error {
			t.Fatal("fallback ran after a successful link")
			return nil
		},
	}, src, dst))
	requireProxyEntries(t, filepath.Dir(dst), "proxy")
}

func TestCreateLinkSkipsSameSource(t *testing.T) {
	for _, kind := range []string{"same-path", "path-alias", "hardlink", "symlink", "source-symlink"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			src := filepath.Join(dir, "source")
			dst := filepath.Join(dir, "proxy")
			require.NoError(t, os.WriteFile(src, []byte("binary"), 0o755))
			switch kind {
			case "same-path":
				dst = src
			case "path-alias":
				dst = dir + string(filepath.Separator) + "." + string(filepath.Separator) + "source"
			case "hardlink":
				if !hardlinksAvailable(t, dir) {
					t.Skip("filesystem policy denies hard links")
				}
				require.NoError(t, os.Link(src, dst))
			case "symlink", "source-symlink":
				if err := os.Symlink(src, dst); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
				if kind == "source-symlink" {
					src, dst = dst, src
				}
			}
			before, err := os.Lstat(dst)
			require.NoError(t, err)
			require.NoError(t, createLinkWith([]linkStrategy{func(string, string) error {
				t.Fatal("an existing same-source entry must not be removed or recreated")
				return nil
			}}, src, dst))
			after, err := os.Lstat(dst)
			require.NoError(t, err)
			assert.True(t, os.SameFile(before, after))
			data, err := os.ReadFile(src)
			require.NoError(t, err)
			assert.Equal(t, []byte("binary"), data)
		})
	}
}

func TestCreateLinkReplacesOldEntry(t *testing.T) {
	for _, kind := range []string{"regular", "symlink", "broken-symlink", "self-loop", "directory-symlink"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			src, dst := filepath.Join(dir, "cjv"), filepath.Join(dir, "cjc")
			oldTarget := filepath.Join(dir, "user-file")
			require.NoError(t, os.WriteFile(src, []byte("new"), 0o755))
			require.NoError(t, os.WriteFile(oldTarget, []byte("untouched"), 0o755))
			if kind == "regular" {
				require.NoError(t, os.WriteFile(dst, []byte("old"), 0o755))
			} else {
				target := oldTarget
				switch kind {
				case "broken-symlink":
					target = filepath.Join(dir, "missing")
				case "self-loop":
					target = dst
				case "directory-symlink":
					target = t.TempDir()
					oldTarget = filepath.Join(target, "user-file")
					require.NoError(t, os.WriteFile(oldTarget, []byte("untouched"), 0o644))
				}
				if err := os.Symlink(target, dst); err != nil {
					t.Skipf("symlinks unavailable: %v", err)
				}
			}
			require.NoError(t, CreateLink(src, dst))
			data, err := os.ReadFile(dst)
			require.NoError(t, err)
			assert.Equal(t, []byte("new"), data)
			data, err = os.ReadFile(oldTarget)
			require.NoError(t, err)
			assert.Equal(t, []byte("untouched"), data, "replacement must not follow the old link")
			requireProxyEntries(t, dir, "cjv", "cjc", "user-file")
		})
	}
}

func TestCreateLinkRejectsDirectoryDestination(t *testing.T) {
	for _, populated := range []bool{false, true} {
		t.Run(map[bool]string{false: "empty", true: "populated"}[populated], func(t *testing.T) {
			dir := t.TempDir()
			src, dst := filepath.Join(dir, "source"), filepath.Join(dir, "proxy")
			require.NoError(t, os.WriteFile(src, []byte("binary"), 0o755))
			require.NoError(t, os.Mkdir(dst, 0o755))
			if populated {
				require.NoError(t, os.WriteFile(filepath.Join(dst, "owned"), []byte("preserve"), 0o644))
			}
			require.Error(t, createLinkWith([]linkStrategy{func(string, string) error {
				t.Fatal("must not attempt replacement after a removal error")
				return nil
			}}, src, dst))
			assert.DirExists(t, dst)
			if populated {
				data, err := os.ReadFile(filepath.Join(dst, "owned"))
				require.NoError(t, err)
				assert.Equal(t, []byte("preserve"), data)
			}
		})
	}
}

func TestCreateLinkRejectsMissingSourceBeforeRemovingDestination(t *testing.T) {
	dir := t.TempDir()
	dst := filepath.Join(dir, "proxy")
	require.NoError(t, os.WriteFile(dst, []byte("old"), 0o755))
	require.ErrorIs(t, CreateLink(filepath.Join(dir, "missing"), dst), os.ErrNotExist)
	data, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, []byte("old"), data)
	require.ErrorIs(t, CreateLink(filepath.Join(dir, "missing"), filepath.Join(dir, "missing")), os.ErrNotExist)
}

func requireProxyEntries(t *testing.T, dir string, want ...string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name()
	}
	assert.ElementsMatch(t, want, names, "proxy installation must not leave scratch paths")
}

// --- Tests merged from copy_file_test.go ---

// CopyFile remains available for managed executable and SDK file copying;
// proxy links do not use it as a fallback.

func TestCopyFile_PreservesContent(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "original.bin")
	dst := filepath.Join(dir, "copy.bin")

	data := []byte("#!/usr/bin/env cjc\nprint(\"hello\")\n")
	require.NoError(t, os.WriteFile(src, data, 0o644))

	require.NoError(t, CopyFile(src, dst, 0o755))

	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, data, got, "copied file content must match source exactly")
}

func TestCopyFile_FailsOnMissingSource(t *testing.T) {
	dir := t.TempDir()
	err := CopyFile(filepath.Join(dir, "nonexistent"), filepath.Join(dir, "dst"), 0o644)
	assert.Error(t, err, "should fail when source file does not exist")
}

func TestCopyFile_LargeFile(t *testing.T) {
	// SDK binaries can be tens of megabytes; verify copy works for non-trivial sizes.
	dir := t.TempDir()
	src := filepath.Join(dir, "large.bin")
	dst := filepath.Join(dir, "large_copy.bin")

	data := make([]byte, 1<<16) // 64 KB
	for i := range data {
		data[i] = byte(i % 251) // prime modulus to avoid patterns
	}
	require.NoError(t, os.WriteFile(src, data, 0o644))

	require.NoError(t, CopyFile(src, dst, 0o755))

	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, data, got)
}
