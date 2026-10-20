package resolve

import (
	"context"
	"os"

	"github.com/Zxilly/cjv/internal/env"
)

// Runtime pairs the active toolchain selection with the environment derived
// from it. env.Runtime stays independent of selection and auto-installation.
type Runtime struct {
	env.Runtime
	Active ActiveToolchain
}

// ActiveRuntime resolves the active toolchain (installing missing
// requirements according to auto_install) and derives its environment.
func ActiveRuntime(ctx context.Context, tcOverride string) (Runtime, error) {
	active, err := Active(ctx, tcOverride)
	if err != nil {
		return Runtime{}, err
	}
	return runtimeFor(active)
}

// ActiveTargetRuntime is ActiveRuntime for a cross-compilation target.
func ActiveTargetRuntime(ctx context.Context, tcOverride, target string) (Runtime, error) {
	active, err := ActiveTarget(ctx, tcOverride, target)
	if err != nil {
		return Runtime{}, err
	}
	return runtimeFor(active)
}

// ActiveRuntimeEnv resolves the active toolchain from context and builds
// the full environment needed to run compiled Cangjie binaries.
// tcOverride is the optional +toolchain argument (empty string to auto-resolve).
func ActiveRuntimeEnv(ctx context.Context, tcOverride string) ([]string, error) {
	rt, err := ActiveRuntime(ctx, tcOverride)
	if err != nil {
		return nil, err
	}
	return rt.ProxyEnv(os.Environ(), 0), nil
}

func runtimeFor(active ActiveToolchain) (Runtime, error) {
	rt, err := env.RuntimeForToolchain(active.Dir, active.Name)
	if err != nil {
		return Runtime{}, err
	}
	return Runtime{Runtime: rt, Active: active}, nil
}
