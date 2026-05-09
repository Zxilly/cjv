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

	assert.Contains(t, cfg.PathPrepend, filepath.Join(sdk, "bin"))
	assert.Contains(t, cfg.PathPrepend, filepath.Join(sdk, "tools", "bin"))
	assert.NotContains(t, cfg.PathPrepend, filepath.Join(sdk, "tools", "lib"),
		"missing tools/lib should not be added")
}

func TestDeriveToolchainEnv_NoBackendDirReturnsNoLibPaths(t *testing.T) {
	sdk := t.TempDir()
	writeDir(t, sdk, "bin")

	cfg := DeriveToolchainEnv(sdk)

	for _, p := range cfg.PathPrepend {
		assert.NotContains(t, p, "_cjnative")
		assert.NotContains(t, p, "_llvm")
	}
	for _, p := range cfg.LibraryPathPrepend {
		assert.NotContains(t, p, "_cjnative")
		assert.NotContains(t, p, "_llvm")
	}
}

func TestHostBackendDirForArch_DetectsCjnative(t *testing.T) {
	sdk := t.TempDir()
	backend := "linux_x86_64_cjnative"
	writeDir(t, sdk, "runtime", "lib", backend)

	assert.Equal(t, backend, hostBackendDirForArch(sdk, "x86_64"))
}

func TestHostBackendDirForArch_DetectsLlvm(t *testing.T) {
	sdk := t.TempDir()
	backend := "linux_x86_64_llvm"
	writeDir(t, sdk, "runtime", "lib", backend)

	assert.Equal(t, backend, hostBackendDirForArch(sdk, "x86_64"))
}

func TestHostBackendDirForArch_IgnoresCrossTargets(t *testing.T) {
	sdk := t.TempDir()
	host := "linux_x86_64_cjnative"
	writeDir(t, sdk, "runtime", "lib", host)
	// Cross-compile runtimes share the parent directory but have an extra
	// segment between OS and arch (e.g. linux_ohos_aarch64_cjnative).
	writeDir(t, sdk, "runtime", "lib", "linux_ohos_aarch64_cjnative")

	assert.Equal(t, host, hostBackendDirForArch(sdk, "x86_64"))
}

func TestHostBackendDirForArch_OSPrefixAgnostic(t *testing.T) {
	// The host arch suffix (x86_64 / aarch64) is what we match on, not the
	// OS prefix — Cangjie's archive filenames use "mac" while Go uses
	// "darwin", and we shouldn't have to know which one ends up in the
	// runtime/lib directory name.
	sdk := t.TempDir()
	custom := "someos_x86_64_cjnative"
	writeDir(t, sdk, "runtime", "lib", custom)

	assert.Equal(t, custom, hostBackendDirForArch(sdk, "x86_64"))
}

func TestHostBackendDirForArch_PrefersCjnativeOverLlvm(t *testing.T) {
	sdk := t.TempDir()
	cj := "linux_x86_64_cjnative"
	writeDir(t, sdk, "runtime", "lib", cj)
	writeDir(t, sdk, "runtime", "lib", "linux_x86_64_llvm")

	assert.Equal(t, cj, hostBackendDirForArch(sdk, "x86_64"))
}

func TestHostArchMapsGoArch(t *testing.T) {
	assert.Equal(t, "x86_64", hostArchName("amd64"))
	assert.Equal(t, "aarch64", hostArchName("arm64"))
	assert.Equal(t, "riscv64", hostArchName("riscv64"))
	assert.Equal(t, hostArchName(runtime.GOARCH), hostArch())
}
