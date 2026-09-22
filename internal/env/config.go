package env

import (
	"path/filepath"
	"runtime"
	"slices"

	"github.com/Zxilly/cjv/internal/config"
)

// EnvConfig describes the SDK's environment contributions, before they are
// combined with a caller's environment.
type EnvConfig struct {
	Vars               map[string]string
	PathPrepend        []string
	PathAppend         []string
	LibraryPathPrepend []string
}

// ComponentEnvProvider injects env vars contributed by installed components.
// Passed in by callers so the env package does not need to import component.
type ComponentEnvProvider func(vars map[string]string, tcDir string)

// NewEnvConfig returns an initialized empty EnvConfig.
func NewEnvConfig() *EnvConfig {
	return &EnvConfig{Vars: make(map[string]string)}
}

// LoadToolchainEnv computes the runtime environment for the SDK installed
// at tcDir. The configuration is derived from the on-disk layout (no
// envsetup script execution), with component-contributed vars layered on top.
// It does not capture the process's PATH, library search path, or SDKROOT.
func LoadToolchainEnv(tcDir string, componentEnv ComponentEnvProvider) *EnvConfig {
	cfg := DeriveToolchainEnv(tcDir)
	applyComponentEnv(cfg, tcDir, componentEnv)
	return cfg
}

// environmentContributions returns an independent view of the SDK's changes.
// The base is used only to decide which platform defaults are needed; its
// values are never included in the returned contributions.
func environmentContributions(cfg *EnvConfig, baseEnv []string) Contributions {
	result := Contributions{
		Vars:           make(map[string]string),
		LibraryPathKey: libraryPathKey(runtime.GOOS),
	}
	if cfg != nil {
		result.PathPrepend = slices.Clone(cfg.PathPrepend)
		result.PathAppend = slices.Clone(cfg.PathAppend)
		result.LibraryPathPrepend = libraryPathEntries(cfg)
		for key, value := range cfg.Vars {
			canonical := canonicalEnvKey(key)
			// Selection and recursion belong to proxy policy, not the SDK.
			if key == "" || canonical == canonicalEnvKey(config.EnvToolchain) || canonical == canonicalEnvKey(config.EnvRecursionCount) {
				continue
			}
			if key == result.LibraryPathKey {
				// A component's library paths also augment the caller's paths.
				result.LibraryPathPrepend = append(result.LibraryPathPrepend, filepath.SplitList(value)...)
				continue
			}
			result.Vars[key] = value
		}
	}
	applyPlatformVars(result.Vars, baseEnv)
	return result
}

func applyComponentEnv(cfg *EnvConfig, tcDir string, componentEnv ComponentEnvProvider) {
	if componentEnv == nil {
		return
	}
	if cfg.Vars == nil {
		cfg.Vars = make(map[string]string)
	}
	componentEnv(cfg.Vars, tcDir)
}
