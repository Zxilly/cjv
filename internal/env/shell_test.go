package env

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddToShellConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell config tests are for Unix")
	}
	tmp := t.TempDir()
	rcFile := filepath.Join(tmp, ".bashrc")
	os.WriteFile(rcFile, []byte("# existing content\n"), 0o644)

	require.NoError(t, AddPathToShellConfig(rcFile, "/home/user/.cjv/bin"))

	content, _ := os.ReadFile(rcFile)
	assert.Contains(t, string(content), "# cjv (managed by cjv, do not edit)")
	assert.Contains(t, string(content), `/home/user/.cjv/bin`)
	assert.Contains(t, string(content), `export PATH=`)
	assert.Contains(t, string(content), "# cjv end")
}

func TestAddToShellConfigIdempotent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell config tests are for Unix")
	}
	tmp := t.TempDir()
	rcFile := filepath.Join(tmp, ".bashrc")
	os.WriteFile(rcFile, []byte(""), 0o644)

	AddPathToShellConfig(rcFile, "/home/user/.cjv/bin")
	AddPathToShellConfig(rcFile, "/home/user/.cjv/bin")

	content, _ := os.ReadFile(rcFile)
	assert.Equal(t, 1, strings.Count(string(content), "# cjv (managed by cjv, do not edit)"))
}

func TestRemoveFromShellConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell config tests are for Unix")
	}
	tmp := t.TempDir()
	rcFile := filepath.Join(tmp, ".bashrc")
	os.WriteFile(rcFile, []byte("before\n# cjv (managed by cjv, do not edit)\nexport PATH=\"$HOME/.cjv/bin:$PATH\"\n# cjv end\nafter\n"), 0o644)

	require.NoError(t, RemovePathFromShellConfig(rcFile))

	content, _ := os.ReadFile(rcFile)
	assert.NotContains(t, string(content), "cjv")
	assert.Contains(t, string(content), "before")
	assert.Contains(t, string(content), "after")
}

func TestRemoveFromShellConfigNoMarker(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell config tests are for Unix")
	}
	tmp := t.TempDir()
	rcFile := filepath.Join(tmp, ".bashrc")
	original := "# some content\nexport FOO=bar\n"
	os.WriteFile(rcFile, []byte(original), 0o644)

	require.NoError(t, RemovePathFromShellConfig(rcFile))

	content, _ := os.ReadFile(rcFile)
	assert.Equal(t, original, string(content))
}

func TestAddToFishConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell config tests are for Unix")
	}
	tmp := t.TempDir()
	fishConfig := filepath.Join(tmp, "config.fish")
	os.WriteFile(fishConfig, []byte("# fish config\n"), 0o644)

	require.NoError(t, AddPathToFishConfig(fishConfig, "/home/user/.cjv/bin"))

	content, _ := os.ReadFile(fishConfig)
	assert.Contains(t, string(content), "# cjv (managed by cjv, do not edit)")
	assert.Contains(t, string(content), "fish_add_path")
	assert.Contains(t, string(content), "# cjv end")
}

func TestAddPathToShellConfig_CreatesBlock(t *testing.T) {
	rcPath := filepath.Join(t.TempDir(), ".bashrc")
	binDir := "/home/user/.cjv/bin"

	require.NoError(t, AddPathToShellConfig(rcPath, binDir))

	content, err := os.ReadFile(rcPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), binDir)
	assert.Contains(t, string(content), "# cjv")
}

func TestAddPathToShellConfig_Idempotent(t *testing.T) {
	rcPath := filepath.Join(t.TempDir(), ".bashrc")
	binDir := "/home/user/.cjv/bin"

	require.NoError(t, AddPathToShellConfig(rcPath, binDir))
	firstContent, _ := os.ReadFile(rcPath)

	require.NoError(t, AddPathToShellConfig(rcPath, binDir))
	secondContent, _ := os.ReadFile(rcPath)

	assert.Equal(t, firstContent, secondContent,
		"calling AddPathToShellConfig twice should not duplicate the block")
}

func TestAddPathToShellConfig_PreservesExistingContent(t *testing.T) {
	rcPath := filepath.Join(t.TempDir(), ".bashrc")
	existing := "# My custom config\nexport EDITOR=vim\n"
	require.NoError(t, utils.WriteFileAtomic(rcPath, []byte(existing), 0o644))

	require.NoError(t, AddPathToShellConfig(rcPath, "/cjv/bin"))

	content, _ := os.ReadFile(rcPath)
	assert.Contains(t, string(content), "export EDITOR=vim",
		"existing content must be preserved")
	assert.Contains(t, string(content), "/cjv/bin")
}

func TestRemovePathFromShellConfig_RemovesBlock(t *testing.T) {
	rcPath := filepath.Join(t.TempDir(), ".bashrc")
	binDir := "/home/user/.cjv/bin"

	require.NoError(t, AddPathToShellConfig(rcPath, binDir))
	require.NoError(t, RemovePathFromShellConfig(rcPath))

	content, _ := os.ReadFile(rcPath)
	assert.NotContains(t, string(content), binDir,
		"bin dir should be removed")
	assert.NotContains(t, string(content), "# cjv",
		"marker comments should be removed")
}

func TestRemovePathFromShellConfig_NoOpWhenNoMarker(t *testing.T) {
	rcPath := filepath.Join(t.TempDir(), ".bashrc")
	original := "export PATH=$PATH:/usr/local/bin\n"
	require.NoError(t, os.WriteFile(rcPath, []byte(original), 0o644))

	require.NoError(t, RemovePathFromShellConfig(rcPath))

	content, _ := os.ReadFile(rcPath)
	assert.Equal(t, original, string(content),
		"file without marker should be unchanged")
}

func TestRemovePathFromShellConfig_MissingFileIsNotError(t *testing.T) {
	err := RemovePathFromShellConfig(filepath.Join(t.TempDir(), "nonexistent"))
	assert.NoError(t, err, "removing from non-existent file should not error")
}

func TestAddPathToFishConfig_CreatesBlock(t *testing.T) {
	fishPath := filepath.Join(t.TempDir(), "config.fish")
	binDir := "/home/user/.cjv/bin"

	require.NoError(t, AddPathToFishConfig(fishPath, binDir))

	content, err := os.ReadFile(fishPath)
	require.NoError(t, err)
	assert.Contains(t, string(content), "fish_add_path")
	assert.Contains(t, string(content), binDir)
}

func TestShellConfigPaths_ReturnsNonNil(t *testing.T) {
	posix, fish := ShellConfigPaths()
	assert.NotNil(t, posix, "posix config paths should not be nil")
	// fish may be empty if fish is not installed, which is fine
	_ = fish
}

func TestFilePermOrDefault_NonExistentReturnsDefault(t *testing.T) {
	perm := filePermOrDefault(filepath.Join(t.TempDir(), "no-file"), 0o644)
	assert.Equal(t, os.FileMode(0o644), perm)
}

func TestFilePermOrDefault_ExistingReturnsActual(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.sh")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh"), 0o755))

	perm := filePermOrDefault(path, 0o644)
	assert.NotZero(t, perm)
}
