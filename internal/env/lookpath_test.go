package env

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookPathInEnv_ExtensionHandling(t *testing.T) {
	dir := t.TempDir()
	environ := []string{"PATH=" + dir}

	if runtime.GOOS == "windows" {
		// A same-named extensionless file must NOT shadow the real .exe.
		require.NoError(t, os.WriteFile(filepath.Join(dir, "tool"), []byte("data"), 0o644))
		exe := filepath.Join(dir, "tool.exe")
		require.NoError(t, os.WriteFile(exe, []byte("MZ"), 0o644))
		environ = append(environ, "PATHEXT=.COM;.EXE;.BAT;.CMD")

		got, found := LookPathInEnv("tool", environ)
		require.True(t, found)
		assert.Equal(t, exe, got)
	} else {
		exe := filepath.Join(dir, "tool")
		require.NoError(t, os.WriteFile(exe, []byte("#!/bin/sh\n"), 0o755))

		got, found := LookPathInEnv("tool", environ)
		require.True(t, found)
		assert.Equal(t, exe, got)
	}

	// A command not present on the env PATH is reported as not found.
	got, found := LookPathInEnv("definitely-not-present-xyz", environ)
	assert.False(t, found)
	assert.Equal(t, "definitely-not-present-xyz", got)
}

func TestLookPathInEnv_WindowsDottedCommand(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("PATHEXT lookup is Windows-specific")
	}
	dir := t.TempDir()
	exe := filepath.Join(dir, "tool.v1.exe")
	require.NoError(t, os.WriteFile(exe, []byte("MZ"), 0o644))
	environ := []string{"PATH=" + dir, "PATHEXT=.EXE;.CMD"}
	got, found := LookPathInEnv("tool.v1", environ)
	require.True(t, found, "a dot in a command name must not disable PATHEXT lookup")
	assert.Equal(t, exe, got)

	// An explicitly named executable still takes precedence over appended extensions.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tool.v1.exe.cmd"), []byte("stub"), 0o644))
	got, found = LookPathInEnv("tool.v1.exe", environ)
	require.True(t, found)
	assert.Equal(t, exe, got)
}
