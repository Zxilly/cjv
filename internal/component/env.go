package component

import (
	"path/filepath"
)

// stdx exposes its dynamic / static library directories to proxied tools
// through these env vars. Read by the runtime when locating extension libs.
const (
	EnvStdxDynamic = "CANGJIE_STDX_PATH_DYNAMIC"
	EnvStdxStatic  = "CANGJIE_STDX_PATH_STATIC"
)

// ApplyEnv satisfies env.ComponentEnvHook: each installed component contributes
// the runtime environment variables declared in its Spec.EnvVars (currently
// only stdx). Driven by Spec rather than special-cased per component name.
//
// This is on the proxy hot path (called for every proxied tool invocation), so
// it iterates the spec table directly — skipping components with no EnvVars
// before any filesystem check — and computes the install roots lazily, only
// once an env-contributing component is found installed.
func ApplyEnv(vars map[string]string, tcDir string) {
	if vars == nil {
		return
	}
	var roots Roots
	rootsReady := false
	for name, spec := range specs {
		if len(spec.EnvVars) == 0 || !IsInstalled(tcDir, name) {
			continue
		}
		if !rootsReady {
			var err error
			if roots, err = RootsFor(filepath.Base(tcDir)); err != nil {
				return
			}
			rootsReady = true
		}
		root := spec.InstallRoot(roots)
		for envName, subdir := range spec.EnvVars {
			vars[envName] = filepath.Join(root, subdir)
		}
	}
}
