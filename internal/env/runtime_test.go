package env_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimeHonorsBaseLibraryPath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows loads SDK libraries through PATH")
	}
	config.IsolateForTest(t, t.TempDir())
	key := "LD_LIBRARY_PATH"
	if runtime.GOOS == "darwin" {
		key = "DYLD_LIBRARY_PATH"
	}
	t.Setenv(key, "/construction-process-library")
	sdk := t.TempDir()
	libraryDir := filepath.Join(sdk, "tools", "lib")
	require.NoError(t, os.MkdirAll(libraryDir, 0o755))
	rt, err := env.RuntimeForToolchain(sdk, "review-sdk", nil)
	require.NoError(t, err)

	for _, mode := range []struct {
		name  string
		build func([]string) []string
	}{
		{"proxy", func(base []string) []string { return rt.ProxyEnv(base, 3) }},
		{"toolchain", rt.ToolchainEnv},
		{"shell", func(base []string) []string {
			// macOS strips DYLD_* when starting protected system programs.
			// Establish the base inside the shell and observe it with a builtin
			// so this tests the emitted script, not the OS's exec filtering.
			script := "unset " + key + "\n"
			if value, ok := env.LookupValue(base, key); ok {
				quoted := "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
				script = "export " + key + "=" + quoted + "\n"
			}
			script += rt.ShellScript(base, env.ShellPosix) + "printf '%s' \"$" + key + "\""
			cmd := exec.Command("/bin/sh", "-c", script)
			cmd.Env = append([]string{}, base...)
			out, err := cmd.Output()
			require.NoError(t, err)
			return []string{key + "=" + string(out)}
		}},
	} {
		t.Run(mode.name, func(t *testing.T) {
			base := []string{"PATH=/caller/bin", key + "=/caller-library"}
			got := mode.build(base)
			value, ok := env.LookupValue(got, key)
			require.True(t, ok)
			assert.Equal(t, libraryDir+string(os.PathListSeparator)+"/caller-library", value)
			assert.Equal(t, []string{"PATH=/caller/bin", key + "=/caller-library"}, base)

			// Reusing the Runtime must neither capture the previous base nor
			// duplicate SDK entries already present in a prepared environment.
			repeated, _ := env.LookupValue(mode.build(got), key)
			assert.Equal(t, value, repeated)
			emptyBase, _ := env.LookupValue(mode.build(nil), key)
			assert.Equal(t, libraryDir, emptyBase)
		})
	}
}

func TestRuntimeContributionsExcludeInheritedEnvironment(t *testing.T) {
	config.IsolateForTest(t, t.TempDir())
	sdk := t.TempDir()
	bin := filepath.Join(sdk, "bin")
	libraryDir := filepath.Join(sdk, "tools", "lib")
	require.NoError(t, os.MkdirAll(bin, 0o755))
	require.NoError(t, os.MkdirAll(libraryDir, 0o755))
	rt, err := env.RuntimeForToolchain(sdk, "lts-1.0.5", nil)
	require.NoError(t, err)
	base := []string{
		"PATH=/caller/bin", "OTHER=keep", "CJV_TOOLCHAIN=old", "CJV_RECURSION_COUNT=5",
		"LD_LIBRARY_PATH=/caller-library", "DYLD_LIBRARY_PATH=/caller-library",
	}
	contributed := rt.Contributions(base)
	assert.Equal(t, sdk, contributed.Vars["CANGJIE_HOME"])
	assert.NotContains(t, contributed.Vars, "OTHER")
	assert.NotContains(t, contributed.Vars, "CJV_TOOLCHAIN")
	assert.NotContains(t, contributed.Vars, "CJV_RECURSION_COUNT")
	assert.NotContains(t, contributed.Vars, "LD_LIBRARY_PATH")
	assert.NotContains(t, contributed.Vars, "DYLD_LIBRARY_PATH")
	assert.Contains(t, contributed.PathPrepend, bin)
	assert.NotContains(t, contributed.PathPrepend, "/caller/bin")
	if runtime.GOOS == "windows" {
		assert.Empty(t, contributed.LibraryPathKey)
		assert.Empty(t, contributed.LibraryPathPrepend)
		assert.Contains(t, contributed.PathPrepend, libraryDir)
	} else {
		assert.Equal(t, []string{libraryDir}, contributed.LibraryPathPrepend)
	}

	// Editing a rendered view must not change a later environment or view.
	contributed.Vars["CANGJIE_HOME"] = "/changed"
	contributed.PathPrepend[0] = "/changed"
	if len(contributed.LibraryPathPrepend) > 0 {
		contributed.LibraryPathPrepend[0] = "/changed"
	}
	again := rt.Contributions(base)
	assert.Equal(t, sdk, again.Vars["CANGJIE_HOME"])
	assert.NotContains(t, again.PathPrepend, "/changed")
	assert.NotContains(t, again.LibraryPathPrepend, "/changed")
	prepared := rt.ToolchainEnv(base)
	assert.Empty(t, rt.ShellScript(prepared, env.DefaultShellType()))
}

func TestRuntimePreservesUnixPathKeyCasing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("environment keys are case insensitive on Windows")
	}
	config.IsolateForTest(t, t.TempDir())
	sdk := t.TempDir()
	bin := filepath.Join(sdk, "bin")
	require.NoError(t, os.MkdirAll(bin, 0o755))
	rt, err := env.RuntimeForToolchain(sdk, "lts-1.0.5", nil)
	require.NoError(t, err)
	base := []string{"PATH=/caller/bin", "path=/unrelated-variable"}
	for _, got := range [][]string{rt.ProxyEnv(base, 0), rt.ToolchainEnv(base)} {
		path, _ := env.LookupValue(got, "PATH")
		assert.Contains(t, filepath.SplitList(path), bin)
		assert.Contains(t, filepath.SplitList(path), "/caller/bin")
		lowercase, _ := env.LookupValue(got, "path")
		assert.Equal(t, "/unrelated-variable", lowercase)
	}
}

func TestRuntimePreservesProxyAndToolchainEnvironmentModes(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	sdk := t.TempDir()
	bin := filepath.Join(sdk, "bin")
	require.NoError(t, os.MkdirAll(bin, 0o755))
	rt, err := env.RuntimeForToolchain(sdk, "lts-1.0.5", func(vars map[string]string, dir string) {
		vars["CANGJIE_STDX_PATH_DYNAMIC"] = filepath.Join(dir, "stdx", "dynamic")
	})
	require.NoError(t, err)

	base := []string{"PATH=/caller/bin", "OTHER=keep", "=C:=C:\\work"}
	for _, mode := range []struct {
		name  string
		build func([]string) []string
	}{
		{"proxy", func(base []string) []string { return rt.ProxyEnv(base, 3) }},
		{"toolchain", rt.ToolchainEnv},
	} {
		t.Run(mode.name, func(t *testing.T) {
			got := mode.build(base)
			path, _ := env.LookupValue(got, "PATH")
			parts := filepath.SplitList(path)
			assert.Contains(t, parts, "/caller/bin")
			assert.Contains(t, got, "OTHER=keep")
			assert.Contains(t, got, "=C:=C:\\work")
			assert.Contains(t, got, "CANGJIE_STDX_PATH_DYNAMIC="+filepath.Join(sdk, "stdx", "dynamic"))
			if mode.name == "proxy" {
				assert.Equal(t, filepath.Join(home, "bin"), parts[0])
				assert.Contains(t, got, "CJV_RECURSION_COUNT=4")
				assert.Contains(t, got, "CJV_TOOLCHAIN=lts-1.0.5")
			} else {
				assert.Equal(t, bin, parts[0])
				_, recursionSet := env.LookupValue(got, "CJV_RECURSION_COUNT")
				_, toolchainSet := env.LookupValue(got, "CJV_TOOLCHAIN")
				assert.False(t, recursionSet)
				assert.False(t, toolchainSet)
			}
		})
	}
}

func TestRuntimePreservesWindowsEnvironmentCasing(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("environment keys are case sensitive on Unix")
	}
	config.IsolateForTest(t, t.TempDir())
	rt, err := env.RuntimeForToolchain(t.TempDir(), "lts-1.0.5", nil)
	require.NoError(t, err)
	base := []string{"Path=C:\\old", "path=C:\\caller", "cangjie_home=C:\\old-sdk", "=D:=D:\\work"}
	for _, got := range [][]string{rt.ProxyEnv(base, 0), rt.ToolchainEnv(base)} {
		var pathCount, homeCount int
		for _, entry := range got {
			key, _, _ := strings.Cut(entry, "=")
			if strings.EqualFold(key, "PATH") {
				pathCount++
			}
			if strings.EqualFold(key, "CANGJIE_HOME") {
				homeCount++
			}
		}
		assert.Equal(t, 1, pathCount)
		assert.Equal(t, 1, homeCount)
		path, _ := env.LookupValue(got, "PATH")
		assert.Contains(t, filepath.SplitList(path), "C:\\caller")
		assert.NotContains(t, filepath.SplitList(path), "C:\\old")
		assert.Contains(t, got, "=D:=D:\\work")
	}
}
