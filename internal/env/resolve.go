package env

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// ResolveRuntimeEnv resolves the active toolchain from context and builds
// the full environment needed to run compiled Cangjie binaries.
// tcOverride is the optional +toolchain argument (empty string to auto-resolve).
func ResolveRuntimeEnv(ctx context.Context, tcOverride string) ([]string, error) {
	var tcName string
	if tcOverride != "" {
		tcName = tcOverride
	} else if envTC := os.Getenv(config.EnvToolchain); envTC != "" {
		tcName = envTC
	} else {
		sf, err := config.DefaultSettingsFile()
		if err != nil {
			return nil, err
		}
		settings, err := sf.Load()
		if err != nil {
			return nil, err
		}
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get working directory: %w", err)
		}
		resolved, _, err := config.ResolveToolchain(settings, cwd)
		if err != nil {
			return nil, err
		}
		tcName = resolved
	}

	parsed, err := toolchain.ParseToolchainName(tcName)
	if err != nil {
		return nil, err
	}
	tcDir, err := toolchain.FindInstalled(parsed)
	if err != nil {
		return nil, fmt.Errorf("toolchain '%s' is not installed, run: cjv install %s", tcName, tcName)
	}

	envCfg := LoadToolchainEnv(ctx, tcDir)

	binDir, err := config.BinDir()
	if err != nil {
		return nil, fmt.Errorf("failed to determine bin directory: %w", err)
	}

	proxyEnv := BuildProxyEnv(os.Environ(), ProxyEnvContext{
		Cfg:             envCfg,
		CjvBinDir:       binDir,
		ToolchainBinDir: filepath.Join(tcDir, "bin"),
		Recursion:       0,
		ToolchainName:   filepath.Base(tcDir),
	})

	return proxyEnv, nil
}
