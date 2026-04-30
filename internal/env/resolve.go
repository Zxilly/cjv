package env

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/resolve"
)

// ResolveRuntimeEnv resolves the active toolchain from context and builds
// the full environment needed to run compiled Cangjie binaries.
// tcOverride is the optional +toolchain argument (empty string to auto-resolve).
func ResolveRuntimeEnv(ctx context.Context, tcOverride string) ([]string, error) {
	active, err := resolve.Active(ctx, tcOverride)
	if err != nil {
		return nil, err
	}

	envCfg := LoadToolchainEnv(ctx, active.Dir)

	binDir, err := config.BinDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine bin directory: %w", err)
	}

	proxyEnv := BuildProxyEnv(os.Environ(), ProxyEnvContext{
		Cfg:             envCfg,
		CjvBinDir:       binDir,
		ToolchainBinDir: filepath.Join(active.Dir, "bin"),
		Recursion:       0,
		ToolchainName:   active.Name,
	})

	return proxyEnv, nil
}
