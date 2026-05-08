package env

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// hostArch maps runtime.GOARCH to the SDK's arch suffix used in
// runtime/lib subdirectory names (e.g. "x86_64" in "windows_x86_64_cjnative").
func hostArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	}
	return runtime.GOARCH
}

// hostBackendDir returns the SDK's host backend subdirectory name (e.g.
// "windows_x86_64_cjnative") by scanning runtime/lib for an entry shaped
// like "<os>_<host_arch>_<backend>". The host arch component is what we
// match on — cross-target directories have an extra segment between the OS
// and the arch (e.g. "linux_ohos_aarch64_cjnative") so they fall out
// naturally, and we don't have to guess the OS prefix (which differs
// between the manifest's "mac" and Go's "darwin", for example).
//
// Returns an empty string when no matching directory is found.
func hostBackendDir(sdkDir string) string {
	suffixes := []string{
		"_" + hostArch() + "_cjnative",
		"_" + hostArch() + "_llvm",
	}
	entries, err := os.ReadDir(filepath.Join(sdkDir, "runtime", "lib"))
	if err != nil {
		return ""
	}
	for _, suffix := range suffixes {
		for _, e := range entries {
			if !e.IsDir() {
				continue
			}
			name := e.Name()
			if !strings.HasSuffix(name, suffix) {
				continue
			}
			// Host dirs are exactly "<os>_<arch>_<backend>" — three
			// segments where the os has no underscore. Cross-target dirs
			// like "linux_ohos_aarch64_cjnative" carry an extra segment
			// before the arch, so we reject anything with an underscore
			// before the host-arch suffix.
			prefix := strings.TrimSuffix(name, suffix)
			if !strings.Contains(prefix, "_") {
				return name
			}
		}
	}
	return ""
}

// DeriveToolchainEnv computes the runtime environment for a Cangjie SDK
// installed at sdkDir purely from the on-disk layout. The SDK ships static
// envsetup scripts whose only meaningful axis of variation is the
// cjnative-vs-llvm backend directory under runtime/lib — everything else is
// a fixed function of CANGJIE_HOME, so we reproduce the script's effect
// without spawning a shell.
func DeriveToolchainEnv(sdkDir string) *EnvConfig {
	cfg := NewEnvConfig()
	cfg.Vars["CANGJIE_HOME"] = sdkDir

	appendIfDir := func(p string) {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			cfg.PathPrepend.Entries = append(cfg.PathPrepend.Entries, p)
		}
	}

	appendIfDir(filepath.Join(sdkDir, "bin"))
	appendIfDir(filepath.Join(sdkDir, "tools", "bin"))
	appendIfDir(filepath.Join(sdkDir, "tools", "lib"))

	if backendDir := hostBackendDir(sdkDir); backendDir != "" {
		appendIfDir(filepath.Join(sdkDir, "lib", backendDir))
		appendIfDir(filepath.Join(sdkDir, "runtime", "lib", backendDir))
	}

	if runtime.GOOS != "windows" {
		appendIfDir(filepath.Join(sdkDir, "third_party", "llvm", "lldb", "lib"))
	}

	return cfg
}
