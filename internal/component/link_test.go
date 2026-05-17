package component

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeStdxSource builds a directory shaped like an unpacked stdx archive:
// <root>/dynamic/<lib> and <root>/static/<lib>. Returns the root.
func makeStdxSource(t *testing.T, lib string) string {
	t.Helper()
	src := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(src, "dynamic"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(src, "static"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "dynamic", lib), []byte("dynamic-bytes"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(src, "static", lib), []byte("static-bytes"), 0o644))
	return src
}

// linkRoots builds a Roots triple with all three directories rooted in
// fresh temp dirs. Mirrors the real layout enough for Link's purposes
// without involving config.Home().
func linkRoots(t *testing.T) Roots {
	t.Helper()
	return Roots{
		TcDir:   t.TempDir(),
		DocsDir: t.TempDir(),
		StdxDir: t.TempDir(),
	}
}

func linkOK(t *testing.T, roots Roots, name Name, src string, force bool) string {
	t.Helper()
	abs, err := Link(roots, name, src, force)
	require.NoError(t, err)
	return abs
}

// assertSymlinkTo verifies linkPath is a symlink/junction whose readlink
// (or stat-target equality for junctions) points at expectedTarget. On
// Windows os.Readlink works for both symlinks and junctions.
func assertSymlinkTo(t *testing.T, linkPath, expectedTarget string) {
	t.Helper()
	info, err := os.Lstat(linkPath)
	require.NoError(t, err, "lstat %s", linkPath)
	assert.NotZero(t, info.Mode()&os.ModeSymlink|os.ModeIrregular, "expected reparse point at %s", linkPath)

	target, err := os.Readlink(linkPath)
	require.NoError(t, err, "readlink %s", linkPath)
	expClean := filepath.Clean(expectedTarget)
	gotClean := filepath.Clean(target)
	// On Windows junctions, Readlink may return an NT-prefixed path.
	if !assert.True(t, gotClean == expClean || filepath.Base(gotClean) == filepath.Base(expClean),
		"link %s points to %q, expected %q", linkPath, target, expectedTarget) {
		return
	}
}

func TestLink_Stdx_Success(t *testing.T) {
	src := makeStdxSource(t, "libfoo")
	roots := linkRoots(t)

	linkOK(t, roots, Stdx, src, false)

	assert.True(t, IsInstalled(roots.TcDir, Stdx))

	dyn := filepath.Join(roots.StdxDir, "dynamic")
	stat := filepath.Join(roots.StdxDir, "static")
	assertSymlinkTo(t, dyn, filepath.Join(src, "dynamic"))
	assertSymlinkTo(t, stat, filepath.Join(src, "static"))

	// User content is reachable through the link.
	data, err := os.ReadFile(filepath.Join(dyn, "libfoo"))
	require.NoError(t, err)
	assert.Equal(t, "dynamic-bytes", string(data))

	manifest, err := ReadManifest(roots.TcDir, Stdx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"dynamic", "static"}, manifest)
}

func TestLink_Stdx_RelativePathResolvedToAbsolute(t *testing.T) {
	src := makeStdxSource(t, "libfoo")
	roots := linkRoots(t)

	cwd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	parent := filepath.Dir(src)
	require.NoError(t, os.Chdir(parent))
	relSrc := filepath.Base(src)

	linkOK(t, roots, Stdx, relSrc, false)
	assert.True(t, IsInstalled(roots.TcDir, Stdx))
}

func TestLink_RejectsNonStdxComponent(t *testing.T) {
	src := makeStdxSource(t, "libfoo")
	roots := linkRoots(t)

	_, err := Link(roots, Docs, src, false)
	var notSupported *cjverr.ComponentLinkNotSupportedError
	require.ErrorAs(t, err, &notSupported)
	assert.Equal(t, "docs", notSupported.Component)
	assert.False(t, IsInstalled(roots.TcDir, Docs))

	_, err = Link(roots, StdxDocs, src, false)
	require.ErrorAs(t, err, &notSupported)
}

func TestLink_PathDoesNotExist(t *testing.T) {
	roots := linkRoots(t)

	_, err := Link(roots, Stdx, filepath.Join(t.TempDir(), "missing"), false)
	var invalid *cjverr.ComponentLinkInvalidPathError
	require.ErrorAs(t, err, &invalid)
	assert.Contains(t, invalid.Reason, "does not exist")
	assert.False(t, IsInstalled(roots.TcDir, Stdx))
}

func TestLink_PathNotDirectory(t *testing.T) {
	roots := linkRoots(t)
	file := filepath.Join(t.TempDir(), "not-a-dir")
	require.NoError(t, os.WriteFile(file, []byte("x"), 0o644))

	_, err := Link(roots, Stdx, file, false)
	var invalid *cjverr.ComponentLinkInvalidPathError
	require.ErrorAs(t, err, &invalid)
	assert.Contains(t, invalid.Reason, "not a directory")
}

func TestLink_MissingDynamicSubdir(t *testing.T) {
	roots := linkRoots(t)
	src := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(src, "static"), 0o755))

	_, err := Link(roots, Stdx, src, false)
	var invalid *cjverr.ComponentLinkInvalidPathError
	require.ErrorAs(t, err, &invalid)
	assert.Contains(t, invalid.Reason, "dynamic")
}

func TestLink_MissingStaticSubdir(t *testing.T) {
	roots := linkRoots(t)
	src := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(src, "dynamic"), 0o755))

	_, err := Link(roots, Stdx, src, false)
	var invalid *cjverr.ComponentLinkInvalidPathError
	require.ErrorAs(t, err, &invalid)
	assert.Contains(t, invalid.Reason, "static")
}

func TestLink_SubdirIsFileNotDirectory(t *testing.T) {
	roots := linkRoots(t)
	src := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(src, "dynamic"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(src, "static"), []byte("x"), 0o644))

	_, err := Link(roots, Stdx, src, false)
	var invalid *cjverr.ComponentLinkInvalidPathError
	require.ErrorAs(t, err, &invalid)
	assert.Contains(t, invalid.Reason, "static")
}

func TestLink_AlreadyInstalledNoForceFails(t *testing.T) {
	src := makeStdxSource(t, "libfoo")
	roots := linkRoots(t)
	linkOK(t, roots, Stdx, src, false)

	_, err := Link(roots, Stdx, src, false)
	var already *cjverr.ComponentAlreadyInstalledError
	require.ErrorAs(t, err, &already)
}

func TestLink_ForceReplacesPreviousLink(t *testing.T) {
	srcOld := makeStdxSource(t, "old-lib")
	srcNew := makeStdxSource(t, "new-lib")
	roots := linkRoots(t)

	linkOK(t, roots, Stdx, srcOld, false)
	linkOK(t, roots, Stdx, srcNew, true)

	assertSymlinkTo(t, filepath.Join(roots.StdxDir, "dynamic"), filepath.Join(srcNew, "dynamic"))
	_, err := os.Stat(filepath.Join(roots.StdxDir, "dynamic", "new-lib"))
	assert.NoError(t, err)
}

func TestLink_ForceOverDownloadedInstall(t *testing.T) {
	// Simulate a previously downloaded stdx install (real files + manifest).
	src := makeStdxSource(t, "libfoo")
	roots := linkRoots(t)

	stdxRoot := roots.StdxDir
	require.NoError(t, os.MkdirAll(filepath.Join(stdxRoot, "dynamic"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(stdxRoot, "static"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(stdxRoot, "dynamic", "old.so"), []byte("download"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(stdxRoot, "static", "old.a"), []byte("download"), 0o644))
	require.NoError(t, WriteManifest(roots.TcDir, Stdx, []string{"dynamic/old.so", "static/old.a"}))

	linkOK(t, roots, Stdx, src, true)

	// dynamic and static are now symlinks; old downloaded files are gone.
	assertSymlinkTo(t, filepath.Join(stdxRoot, "dynamic"), filepath.Join(src, "dynamic"))
	manifest, err := ReadManifest(roots.TcDir, Stdx)
	require.NoError(t, err)
	assert.ElementsMatch(t, []string{"dynamic", "static"}, manifest)
}

func TestLink_RemovePreservesUserData(t *testing.T) {
	src := makeStdxSource(t, "libfoo")
	roots := linkRoots(t)
	linkOK(t, roots, Stdx, src, false)

	require.NoError(t, Remove(roots, Stdx))

	assert.False(t, IsInstalled(roots.TcDir, Stdx))
	// User's source directory and its files are untouched.
	assert.DirExists(t, src)
	assert.DirExists(t, filepath.Join(src, "dynamic"))
	assert.FileExists(t, filepath.Join(src, "dynamic", "libfoo"))
	assert.FileExists(t, filepath.Join(src, "static", "libfoo"))
	// Symlinks inside cjv's stdx root are gone.
	_, err := os.Lstat(filepath.Join(roots.StdxDir, "dynamic"))
	assert.True(t, os.IsNotExist(err), "dynamic link should be removed, got err=%v", err)
}

func TestLink_RemoveAllStdxRootPreservesUserData(t *testing.T) {
	// Simulates `cjv toolchain uninstall` which RemoveAllRetry's the stdx
	// root: this must remove the symlinks but never follow them into the
	// user's source directory.
	src := makeStdxSource(t, "libfoo")
	roots := linkRoots(t)
	linkOK(t, roots, Stdx, src, false)

	require.NoError(t, os.RemoveAll(roots.StdxDir))

	assert.DirExists(t, src)
	assert.FileExists(t, filepath.Join(src, "dynamic", "libfoo"))
	assert.FileExists(t, filepath.Join(src, "static", "libfoo"))
}

func TestLink_EnvVarsInjected(t *testing.T) {
	src := makeStdxSource(t, "libfoo")

	// ApplyEnv resolves stdx paths via config.StdxDirFor(tcName), which uses
	// CJV_HOME. Set up an isolated home with a toolchain dir mirroring the
	// real layout.
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	tcName := "lts-1.0.5"
	tcDir := filepath.Join(home, "toolchains", tcName)
	require.NoError(t, os.MkdirAll(tcDir, 0o755))

	roots, err := RootsFor(tcName)
	require.NoError(t, err)
	linkOK(t, roots, Stdx, src, false)

	vars := make(map[string]string)
	ApplyEnv(vars, tcDir)
	expectedDyn := filepath.Join(home, "stdx", tcName, "dynamic")
	expectedStatic := filepath.Join(home, "stdx", tcName, "static")
	assert.Equal(t, expectedDyn, vars[EnvStdxDynamic])
	assert.Equal(t, expectedStatic, vars[EnvStdxStatic])

	// Resolving the symlink ultimately reaches the user's library.
	resolved, err := filepath.EvalSymlinks(vars[EnvStdxDynamic])
	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(resolved, "libfoo"))
}
