package env

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeriveToolchainEnvForHostMatchesLinuxEnvsetup(t *testing.T) {
	sdk := t.TempDir()
	home := t.TempDir()
	backend := "linux_x86_64_cjnative"

	bin := writeDir(t, sdk, "bin")
	toolsBin := writeDir(t, sdk, "tools", "bin")
	toolsLib := writeDir(t, sdk, "tools", "lib")
	runtimeLib := writeDir(t, sdk, "runtime", "lib", backend)
	libBackend := writeDir(t, sdk, "lib", backend)
	lldbLib := writeDir(t, sdk, "third_party", "llvm", "lldb", "lib")
	cjpmBin := writeDir(t, home, ".cjpm", "bin")

	cfg := deriveToolchainEnvForHost(sdk, "linux", "amd64", home)

	assert.Equal(t, sdk, cfg.Vars["CANGJIE_HOME"])
	assert.Equal(t, []string{bin, toolsBin}, cfg.PathPrepend)
	assert.Equal(t, []string{cjpmBin}, cfg.PathAppend)
	assert.Equal(t, []string{runtimeLib, toolsLib}, cfg.LibraryPathPrepend)
	assert.NotContains(t, cfg.PathPrepend, toolsLib)
	assert.NotContains(t, cfg.PathPrepend, runtimeLib)
	assert.NotContains(t, cfg.PathPrepend, libBackend)
	assert.NotContains(t, cfg.PathPrepend, lldbLib)
}

func TestDeriveToolchainEnvForHostMatchesDarwinEnvsetup(t *testing.T) {
	sdk := t.TempDir()
	home := t.TempDir()
	// Cangjie ships the macOS runtime under the "mac_*" prefix, not Go's
	// "darwin_*" — hostBackendDirForArch is OS-prefix-agnostic so either
	// works, but the fixture matches reality.
	backend := "mac_aarch64_cjnative"

	bin := writeDir(t, sdk, "bin")
	toolsBin := writeDir(t, sdk, "tools", "bin")
	toolsLib := writeDir(t, sdk, "tools", "lib")
	runtimeLib := writeDir(t, sdk, "runtime", "lib", backend)
	libBackend := writeDir(t, sdk, "lib", backend)
	cjpmBin := writeDir(t, home, ".cjpm", "bin")

	cfg := deriveToolchainEnvForHost(sdk, "darwin", "arm64", home)

	assert.Equal(t, sdk, cfg.Vars["CANGJIE_HOME"])
	assert.Equal(t, []string{bin, toolsBin}, cfg.PathPrepend)
	assert.Equal(t, []string{cjpmBin}, cfg.PathAppend)
	assert.Equal(t, []string{runtimeLib, toolsLib}, cfg.LibraryPathPrepend)
	assert.NotContains(t, cfg.PathPrepend, toolsLib)
	assert.NotContains(t, cfg.PathPrepend, runtimeLib)
	assert.NotContains(t, cfg.PathPrepend, libBackend)
}

func TestDeriveToolchainEnvForHostMatchesWindowsEnvsetup(t *testing.T) {
	sdk := t.TempDir()
	home := t.TempDir()
	backend := "windows_x86_64_cjnative"

	toolsLib := writeDir(t, sdk, "tools", "lib")
	toolsBin := writeDir(t, sdk, "tools", "bin")
	bin := writeDir(t, sdk, "bin")
	libBackend := writeDir(t, sdk, "lib", backend)
	runtimeLib := writeDir(t, sdk, "runtime", "lib", backend)
	cjpmBin := writeDir(t, home, ".cjpm", "bin")

	cfg := deriveToolchainEnvForHost(sdk, "windows", "amd64", home)

	assert.Equal(t, sdk, cfg.Vars["CANGJIE_HOME"])
	assert.Equal(t, []string{toolsLib, toolsBin, bin, libBackend, runtimeLib}, cfg.PathPrepend)
	assert.Equal(t, []string{cjpmBin}, cfg.PathAppend)
	assert.Empty(t, cfg.LibraryPathPrepend)
}

func TestDeriveToolchainEnvForHostAppendsCjpmBinEvenWhenMissing(t *testing.T) {
	sdk := t.TempDir()
	home := t.TempDir()

	cfg := deriveToolchainEnvForHost(sdk, "linux", "amd64", home)

	assert.Equal(t, []string{filepath.Join(home, ".cjpm", "bin")}, cfg.PathAppend)
}

func TestLibraryPathKeyForHost(t *testing.T) {
	assert.Equal(t, "DYLD_LIBRARY_PATH", libraryPathKey("darwin"))
	assert.Equal(t, "LD_LIBRARY_PATH", libraryPathKey("linux"))
	assert.Equal(t, "", libraryPathKey("windows"))
}

func TestOpenHarmonyRuntimeEnvironment(t *testing.T) {
	sdk, home := t.TempDir(), t.TempDir()
	bin := writeDir(t, sdk, "bin")
	tools := writeDir(t, sdk, "tools", "bin")
	libs := writeDir(t, sdk, "tools", "lib")
	runtimeLib := writeDir(t, sdk, "runtime", "lib", "ohos_aarch64_cjnative")
	cfg := deriveToolchainEnvForHost(sdk, "openharmony", "arm64", home)
	assert.Equal(t, "LD_LIBRARY_PATH", libraryPathKey("openharmony"))
	assert.Equal(t, []string{bin, tools}, cfg.PathPrepend)
	assert.Equal(t, []string{runtimeLib, libs}, cfg.LibraryPathPrepend)
	assert.Equal(t, []string{filepath.Join(home, ".cjpm", "bin")}, cfg.PathAppend)
}

func TestNativeOpenHarmonySDKRuntimeDirectory(t *testing.T) {
	for _, arch := range []string{"arm64", "amd64"} {
		t.Run(arch, func(t *testing.T) {
			sdk, home := t.TempDir(), t.TempDir()
			backendArch := hostArchName(arch)
			linuxLib := writeDir(t, sdk, "runtime", "lib", "linux_"+backendArch+"_cjnative")
			ohosLib := writeDir(t, sdk, "runtime", "lib", "linux_ohos_"+backendArch+"_cjnative")
			toolsLib := writeDir(t, sdk, "tools", "lib")

			ohos := deriveToolchainEnvForHost(sdk, "openharmony", arch, home)
			assert.Equal(t, []string{ohosLib, toolsLib}, ohos.LibraryPathPrepend)

			linux := deriveToolchainEnvForHost(sdk, "linux", arch, home)
			assert.Equal(t, []string{linuxLib, toolsLib}, linux.LibraryPathPrepend)
		})
	}
}

func TestLibraryPathEntriesReturnsExistingLibraryPaths(t *testing.T) {
	sdk := t.TempDir()
	runtimeLib := writeDir(t, sdk, "runtime", "lib", "linux_x86_64_cjnative")
	toolsLib := writeDir(t, sdk, "tools", "lib")

	cfg := &EnvConfig{
		LibraryPathPrepend: []string{
			runtimeLib,
			toolsLib,
			filepath.Join(sdk, "missing"),
		},
	}

	assert.Equal(t, []string{runtimeLib, toolsLib}, libraryPathEntries(cfg))
}

func TestApplyDarwinSDKRootSetsMissingValue(t *testing.T) {
	cfg := NewEnvConfig()

	applyDarwinSDKRoot(cfg.Vars, "", func() (string, error) {
		return "/Library/Developer/CommandLineTools/SDKs/MacOSX.sdk", nil
	})

	assert.Equal(t, "/Library/Developer/CommandLineTools/SDKs/MacOSX.sdk", cfg.Vars["SDKROOT"])
}

func TestApplyDarwinSDKRootPreservesExistingValue(t *testing.T) {
	cfg := NewEnvConfig()

	applyDarwinSDKRoot(cfg.Vars, "/custom/sdk", func() (string, error) {
		t.Fatal("lookup should not run when SDKROOT is already set")
		return "", nil
	})

	assert.NotContains(t, cfg.Vars, "SDKROOT")
}
