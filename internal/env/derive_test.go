package env

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeDir(t *testing.T, parts ...string) string {
	t.Helper()
	p := filepath.Join(parts...)
	require.NoError(t, os.MkdirAll(p, 0o755))
	return p
}

func TestDeriveToolchainEnv_SetsCangjieHome(t *testing.T) {
	sdk := t.TempDir()

	cfg := DeriveToolchainEnv(sdk)

	assert.Equal(t, sdk, cfg.Vars["CANGJIE_HOME"])
}

func TestDeriveToolchainEnv_OnlyExistingDirsAdded(t *testing.T) {
	sdk := t.TempDir()
	writeDir(t, sdk, "bin")
	writeDir(t, sdk, "tools", "bin")

	cfg := DeriveToolchainEnv(sdk)

	assert.Contains(t, cfg.PathPrepend.Entries, filepath.Join(sdk, "bin"))
	assert.Contains(t, cfg.PathPrepend.Entries, filepath.Join(sdk, "tools", "bin"))
	assert.NotContains(t, cfg.PathPrepend.Entries, filepath.Join(sdk, "tools", "lib"),
		"missing tools/lib should not be added")
}

func TestDeriveToolchainEnv_DetectsCjnativeBackend(t *testing.T) {
	sdk := t.TempDir()
	backend := runtime.GOOS + "_" + hostArch() + "_cjnative"
	writeDir(t, sdk, "runtime", "lib", backend)
	writeDir(t, sdk, "lib", backend)

	cfg := DeriveToolchainEnv(sdk)

	assert.Contains(t, cfg.PathPrepend.Entries, filepath.Join(sdk, "runtime", "lib", backend))
	assert.Contains(t, cfg.PathPrepend.Entries, filepath.Join(sdk, "lib", backend))
}

func TestDeriveToolchainEnv_DetectsLlvmBackend(t *testing.T) {
	sdk := t.TempDir()
	backend := runtime.GOOS + "_" + hostArch() + "_llvm"
	writeDir(t, sdk, "runtime", "lib", backend)
	writeDir(t, sdk, "lib", backend)

	cfg := DeriveToolchainEnv(sdk)

	assert.Contains(t, cfg.PathPrepend.Entries, filepath.Join(sdk, "runtime", "lib", backend))
	assert.Contains(t, cfg.PathPrepend.Entries, filepath.Join(sdk, "lib", backend))
}

func TestDeriveToolchainEnv_IgnoresCrossTargetRuntimes(t *testing.T) {
	sdk := t.TempDir()
	hostBackend := runtime.GOOS + "_" + hostArch() + "_cjnative"
	writeDir(t, sdk, "runtime", "lib", hostBackend)
	// Cross-compile runtimes share the same parent directory but should
	// not be added to host PATH.
	writeDir(t, sdk, "runtime", "lib", "linux_ohos_aarch64_cjnative")

	cfg := DeriveToolchainEnv(sdk)

	assert.Contains(t, cfg.PathPrepend.Entries, filepath.Join(sdk, "runtime", "lib", hostBackend))
	assert.NotContains(t, cfg.PathPrepend.Entries, filepath.Join(sdk, "runtime", "lib", "linux_ohos_aarch64_cjnative"))
}

func TestDeriveToolchainEnv_OSPrefixAgnostic(t *testing.T) {
	// The host arch suffix (x86_64 / aarch64) is what we match on, not the
	// OS prefix — Cangjie's archive filenames use "mac" while Go uses
	// "darwin", and we shouldn't have to know which one ends up in the
	// runtime/lib directory name.
	sdk := t.TempDir()
	custom := "someos_" + hostArch() + "_cjnative"
	writeDir(t, sdk, "runtime", "lib", custom)
	writeDir(t, sdk, "lib", custom)

	cfg := DeriveToolchainEnv(sdk)

	assert.Contains(t, cfg.PathPrepend.Entries, filepath.Join(sdk, "runtime", "lib", custom))
	assert.Contains(t, cfg.PathPrepend.Entries, filepath.Join(sdk, "lib", custom))
}

func TestDeriveToolchainEnv_PrefersCjnativeOverLlvm(t *testing.T) {
	// When both backends are present the modern cjnative dir should win.
	sdk := t.TempDir()
	cj := runtime.GOOS + "_" + hostArch() + "_cjnative"
	llvm := runtime.GOOS + "_" + hostArch() + "_llvm"
	writeDir(t, sdk, "runtime", "lib", cj)
	writeDir(t, sdk, "runtime", "lib", llvm)

	cfg := DeriveToolchainEnv(sdk)

	assert.Contains(t, cfg.PathPrepend.Entries, filepath.Join(sdk, "runtime", "lib", cj))
	assert.NotContains(t, cfg.PathPrepend.Entries, filepath.Join(sdk, "runtime", "lib", llvm))
}

func TestDeriveToolchainEnv_NoBackendDirReturnsNoLibPaths(t *testing.T) {
	sdk := t.TempDir()
	writeDir(t, sdk, "bin")

	cfg := DeriveToolchainEnv(sdk)

	for _, p := range cfg.PathPrepend.Entries {
		assert.NotContains(t, p, "_cjnative")
		assert.NotContains(t, p, "_llvm")
	}
}

func TestDeriveToolchainEnv_PosixIncludesLldbWhenPresent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("third_party/llvm/lldb/lib is POSIX-only")
	}
	sdk := t.TempDir()
	lldb := writeDir(t, sdk, "third_party", "llvm", "lldb", "lib")

	cfg := DeriveToolchainEnv(sdk)

	assert.Contains(t, cfg.PathPrepend.Entries, lldb)
}

func TestDeriveToolchainEnv_WindowsSkipsLldb(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific")
	}
	sdk := t.TempDir()
	writeDir(t, sdk, "third_party", "llvm", "lldb", "lib")

	cfg := DeriveToolchainEnv(sdk)

	for _, p := range cfg.PathPrepend.Entries {
		assert.NotContains(t, p, filepath.Join("lldb", "lib"))
	}
}
