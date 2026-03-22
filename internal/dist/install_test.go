package dist

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestZip creates a zip archive containing the given files.
func createTestZip(t *testing.T, files map[string]string) string {
	t.Helper()
	tmp := t.TempDir()
	zipPath := filepath.Join(tmp, "test.zip")

	f, err := os.Create(zipPath)
	require.NoError(t, err)
	defer f.Close()

	w := zip.NewWriter(f)
	for name, content := range files {
		fw, err := w.Create(name)
		require.NoError(t, err)
		_, err = fw.Write([]byte(content))
		require.NoError(t, err)
	}
	require.NoError(t, w.Close())

	return zipPath
}

// createTestTarGz creates a tar.gz archive containing the given files.
func createTestTarGz(t *testing.T, files map[string]string) string {
	t.Helper()
	tmp := t.TempDir()
	tgzPath := filepath.Join(tmp, "test.tar.gz")

	f, err := os.Create(tgzPath)
	require.NoError(t, err)
	defer f.Close()

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	for name, content := range files {
		hdr := &tar.Header{
			Name: name,
			Mode: 0o644,
			Size: int64(len(content)),
		}
		require.NoError(t, tw.WriteHeader(hdr))
		_, err = tw.Write([]byte(content))
		require.NoError(t, err)
	}

	require.NoError(t, tw.Close())
	require.NoError(t, gw.Close())

	return tgzPath
}

func TestInstallSDKFlatZip(t *testing.T) {
	// Flat zip — no nested top-level directory
	zipPath := createTestZip(t, map[string]string{
		"bin/cjc":        "cjc binary",
		"tools/bin/cjpm": "cjpm binary",
		"envsetup.sh":    "echo setup",
	})

	destDir := filepath.Join(t.TempDir(), "lts-1.0.5")
	err := InstallSDK(context.Background(), zipPath, destDir)
	require.NoError(t, err)

	assertFileContent(t, filepath.Join(destDir, "bin", "cjc"), "cjc binary")
	assertFileContent(t, filepath.Join(destDir, "tools", "bin", "cjpm"), "cjpm binary")
	assertFileContent(t, filepath.Join(destDir, "envsetup.sh"), "echo setup")
}

func TestInstallSDKNestedZip(t *testing.T) {
	// Single top-level directory zip (should be flattened)
	zipPath := createTestZip(t, map[string]string{
		"Cangjie-1.0.5/bin/cjc":        "cjc binary",
		"Cangjie-1.0.5/tools/bin/cjpm": "cjpm binary",
		"Cangjie-1.0.5/envsetup.sh":    "echo setup",
	})

	destDir := filepath.Join(t.TempDir(), "lts-1.0.5")
	err := InstallSDK(context.Background(), zipPath, destDir)
	require.NoError(t, err)

	// Verify flattening: files should be directly under destDir
	assertFileContent(t, filepath.Join(destDir, "bin", "cjc"), "cjc binary")
	assertFileContent(t, filepath.Join(destDir, "tools", "bin", "cjpm"), "cjpm binary")
	assertFileContent(t, filepath.Join(destDir, "envsetup.sh"), "echo setup")
}

func TestInstallSDKNestedTarGz(t *testing.T) {
	// Single top-level directory tar.gz
	tgzPath := createTestTarGz(t, map[string]string{
		"cangjie/bin/cjc":        "cjc binary",
		"cangjie/tools/bin/cjpm": "cjpm binary",
	})

	destDir := filepath.Join(t.TempDir(), "nightly-1.1.0")
	err := InstallSDK(context.Background(), tgzPath, destDir)
	require.NoError(t, err)

	assertFileContent(t, filepath.Join(destDir, "bin", "cjc"), "cjc binary")
	assertFileContent(t, filepath.Join(destDir, "tools", "bin", "cjpm"), "cjpm binary")
}

func TestInstallSDKMultipleTopLevel(t *testing.T) {
	// Multiple top-level entries — should not be flattened
	zipPath := createTestZip(t, map[string]string{
		"bin/cjc":    "cjc binary",
		"lib/foo.so": "library",
		"README.md":  "readme",
	})

	destDir := filepath.Join(t.TempDir(), "lts-1.0.5")
	err := InstallSDK(context.Background(), zipPath, destDir)
	require.NoError(t, err)

	assertFileContent(t, filepath.Join(destDir, "bin", "cjc"), "cjc binary")
	assertFileContent(t, filepath.Join(destDir, "lib", "foo.so"), "library")
	assertFileContent(t, filepath.Join(destDir, "README.md"), "readme")
}

func TestInstallSDKInvalidArchive(t *testing.T) {
	tmp := t.TempDir()
	badFile := filepath.Join(tmp, "bad.zip")
	require.NoError(t, os.WriteFile(badFile, []byte("not an archive"), 0o644))

	destDir := filepath.Join(t.TempDir(), "bad-toolchain")
	err := InstallSDK(context.Background(), badFile, destDir)
	assert.Error(t, err)
}

func assertFileContent(t *testing.T, path, expected string) {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err, "file should exist: %s", path)
	assert.Equal(t, expected, string(data))
}

// --- Tests merged from copy_install_test.go ---

func TestCopyDir_PreservesNestedStructure(t *testing.T) {
	// SDK archives have a structure like bin/, lib/, include/.
	// The entire tree must be preserved after copy.
	src := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(src, "bin"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "bin", "cjc"), []byte("binary"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(src, "lib"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "lib", "runtime.so"), []byte("lib"), 0o644))

	dst := filepath.Join(t.TempDir(), "sdk")
	require.NoError(t, copyDir(src, dst))

	assert.FileExists(t, filepath.Join(dst, "bin", "cjc"))
	assert.FileExists(t, filepath.Join(dst, "lib", "runtime.so"))

	binContent, _ := os.ReadFile(filepath.Join(dst, "bin", "cjc"))
	assert.Equal(t, "binary", string(binContent))
}

func TestCopyDir_EmptyDirectory(t *testing.T) {
	// An empty source directory should create an empty destination directory.
	src := t.TempDir()
	dst := filepath.Join(t.TempDir(), "empty")

	require.NoError(t, copyDir(src, dst))
	assert.DirExists(t, dst)
}

func TestCopyEntry_RegularFile(t *testing.T) {
	src := filepath.Join(t.TempDir(), "tool.exe")
	require.NoError(t, os.WriteFile(src, []byte("MZ..."), 0o755))

	dst := filepath.Join(t.TempDir(), "tool_copy.exe")
	require.NoError(t, copyEntry(src, dst))

	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	assert.Equal(t, []byte("MZ..."), got)
}

func TestCopyEntry_DirectoryDelegatesToCopyDir(t *testing.T) {
	// When copyEntry encounters a directory, it should recursively copy it.
	src := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(src, "inner.txt"), []byte("data"), 0o644))

	dst := filepath.Join(t.TempDir(), "copied_dir")
	require.NoError(t, copyEntry(src, dst))

	assert.FileExists(t, filepath.Join(dst, "inner.txt"))
}

func TestCopyDir_DeeplyNested(t *testing.T) {
	// SDK may have deep nesting: lib/cangjie/runtime/arch/...
	src := t.TempDir()
	deep := filepath.Join(src, "a", "b", "c")
	require.NoError(t, os.MkdirAll(deep, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(deep, "leaf.txt"), []byte("deep"), 0o644))

	dst := filepath.Join(t.TempDir(), "copy")
	require.NoError(t, copyDir(src, dst))

	got, err := os.ReadFile(filepath.Join(dst, "a", "b", "c", "leaf.txt"))
	require.NoError(t, err)
	assert.Equal(t, "deep", string(got))
}
