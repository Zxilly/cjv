package env

import (
	"context"
	"os"
)

// ResolveRuntimeEnv resolves the active toolchain from context and builds
// the full environment needed to run compiled Cangjie binaries.
// tcOverride is the optional +toolchain argument (empty string to auto-resolve).
// componentEnv may be nil; when non-nil it injects component-contributed vars.
func ResolveRuntimeEnv(ctx context.Context, tcOverride string, componentEnv ComponentEnvProvider) ([]string, error) {
	runtimeEnv, err := ResolveRuntime(ctx, tcOverride, componentEnv)
	if err != nil {
		return nil, err
	}
	return runtimeEnv.ProxyEnv(os.Environ(), 0), nil
}
