package env

// EnvConfig describes the toolchain-contributed environment used by
// BuildProxyEnv: a set of variables to set and a list of directories to
// prepend to PATH.
type EnvConfig struct {
	Vars        map[string]string
	PathPrepend PathPrepend
}

// ComponentEnvProvider injects env vars contributed by installed components.
// Passed in by callers so the env package does not need to import component.
type ComponentEnvProvider func(vars map[string]string, tcDir string)

type PathPrepend struct {
	Entries []string
}

// NewEnvConfig returns an initialized empty EnvConfig.
func NewEnvConfig() *EnvConfig {
	return &EnvConfig{Vars: make(map[string]string)}
}

// LoadToolchainEnv computes the runtime environment for the SDK installed
// at tcDir. The configuration is derived from the on-disk layout (no
// envsetup script execution); component-contributed vars are layered on top
// and the platform's library search path is then merged with the process
// environment.
func LoadToolchainEnv(tcDir string, componentEnv ComponentEnvProvider) *EnvConfig {
	cfg := DeriveToolchainEnv(tcDir)
	EnsureLibraryPath(cfg, tcDir)
	applyComponentEnv(cfg, tcDir, componentEnv)
	return cfg
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
