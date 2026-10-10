package env

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

func hostArchName(goarch string) string {
	switch goarch {
	case "amd64":
		return "x86_64"
	case "arm64":
		return "aarch64"
	}
	return goarch
}

// hostBackendDir finds the native runtime for the host architecture. Most
// SDKs use <os>_<arch>_<backend>; native HarmonyOS SDKs also use linux_ohos
// as the OS prefix, which denotes a cross target on other hosts.
//
// Returns an empty string when no matching directory is found.
func hostBackendDir(sdkDir, goos, arch string) string {
	suffixes := []string{
		"_" + arch + "_cjnative",
		"_" + arch + "_llvm",
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
			prefix := strings.TrimSuffix(name, suffix)
			matchesHost := !strings.Contains(prefix, "_")
			if goos == "openharmony" {
				matchesHost = prefix == "linux_ohos" || prefix == "ohos"
			}
			if matchesHost {
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
	home, _ := os.UserHomeDir()
	return deriveToolchainEnvForHost(sdkDir, runtime.GOOS, runtime.GOARCH, home)
}

func deriveToolchainEnvForHost(sdkDir, goos, goarch, homeDir string) *EnvConfig {
	cfg := NewEnvConfig()
	cfg.Vars["CANGJIE_HOME"] = sdkDir

	appendIfDir := func(entries *[]string, p string) {
		if info, err := os.Stat(p); err == nil && info.IsDir() {
			*entries = append(*entries, p)
		}
	}

	arch := hostArchName(goarch)
	backendDir := hostBackendDir(sdkDir, goos, arch)

	if goos == "windows" {
		appendIfDir(&cfg.PathPrepend, filepath.Join(sdkDir, "tools", "lib"))
		appendIfDir(&cfg.PathPrepend, filepath.Join(sdkDir, "tools", "bin"))
		appendIfDir(&cfg.PathPrepend, filepath.Join(sdkDir, "bin"))
		if backendDir != "" {
			appendIfDir(&cfg.PathPrepend, filepath.Join(sdkDir, "lib", backendDir))
			appendIfDir(&cfg.PathPrepend, filepath.Join(sdkDir, "runtime", "lib", backendDir))
		}
	} else {
		appendIfDir(&cfg.PathPrepend, filepath.Join(sdkDir, "bin"))
		appendIfDir(&cfg.PathPrepend, filepath.Join(sdkDir, "tools", "bin"))
		if backendDir != "" {
			appendIfDir(&cfg.LibraryPathPrepend, filepath.Join(sdkDir, "runtime", "lib", backendDir))
		}
		appendIfDir(&cfg.LibraryPathPrepend, filepath.Join(sdkDir, "tools", "lib"))
	}

	if homeDir != "" {
		cfg.PathAppend = append(cfg.PathAppend, filepath.Join(homeDir, ".cjpm", "bin"))
	}

	return cfg
}
