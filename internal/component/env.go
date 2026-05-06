package component

import (
	"path/filepath"

	"github.com/Zxilly/cjv/internal/config"
)

// stdx exposes its dynamic / static library directories to proxied tools
// through these env vars. Read by the runtime when locating extension libs.
const (
	EnvStdxDynamic = "CANGJIE_STDX_PATH_DYNAMIC"
	EnvStdxStatic  = "CANGJIE_STDX_PATH_STATIC"
)

// ApplyEnv satisfies env.ComponentEnvHook: stdx contributes two paths;
// docs / stdx-docs add nothing.
func ApplyEnv(vars map[string]string, tcDir string) {
	if vars == nil {
		return
	}
	if !IsInstalled(tcDir, Stdx) {
		return
	}
	tcName := filepath.Base(tcDir)
	stdxRoot, err := config.StdxDirFor(tcName)
	if err != nil {
		return
	}
	vars[EnvStdxDynamic] = filepath.Join(stdxRoot, "dynamic")
	vars[EnvStdxStatic] = filepath.Join(stdxRoot, "static")
}
