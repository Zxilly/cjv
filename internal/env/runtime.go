package env

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/resolve"
)

// Runtime is the deep module for Cangjie runtime environment assembly. It
// carries the active toolchain plus the derived SDK environment and exposes
// narrow views for proxy children, direct toolchain execution, and shell output.
type Runtime struct {
	Active          resolve.ActiveToolchain
	Cfg             *EnvConfig
	CjvBinDir       string
	ToolchainBinDir string
}

func ResolveRuntime(ctx context.Context, tcOverride string, componentEnv ComponentEnvProvider) (Runtime, error) {
	active, err := resolve.Active(ctx, tcOverride)
	if err != nil {
		return Runtime{}, err
	}
	return runtimeForActive(active, componentEnv)
}

func ResolveTargetRuntime(ctx context.Context, tcOverride, target string, componentEnv ComponentEnvProvider) (Runtime, error) {
	active, err := resolve.ActiveTarget(ctx, tcOverride, target)
	if err != nil {
		return Runtime{}, err
	}
	return runtimeForActive(active, componentEnv)
}

func RuntimeForToolchain(dir, name string, componentEnv ComponentEnvProvider) (Runtime, error) {
	return runtimeForActive(resolve.ActiveToolchain{Dir: dir, Name: name}, componentEnv)
}

func runtimeForActive(active resolve.ActiveToolchain, componentEnv ComponentEnvProvider) (Runtime, error) {
	cfg := LoadToolchainEnv(active.Dir, componentEnv)
	binDir, err := config.BinDir()
	if err != nil {
		return Runtime{}, fmt.Errorf("failed to determine bin directory: %w", err)
	}
	return Runtime{
		Active:          active,
		Cfg:             cfg,
		CjvBinDir:       binDir,
		ToolchainBinDir: filepath.Join(active.Dir, "bin"),
	}, nil
}

func (r Runtime) ProxyEnv(baseEnv []string, recursion int) []string {
	return BuildProxyEnv(baseEnv, ProxyEnvContext{
		Cfg:             r.Cfg,
		CjvBinDir:       r.CjvBinDir,
		ToolchainBinDir: r.ToolchainBinDir,
		Recursion:       recursion,
		ToolchainName:   r.Active.Name,
	})
}

func (r Runtime) ToolchainEnv(baseEnv []string) []string {
	return BuildToolchainEnv(baseEnv, r.Cfg)
}

// ToolBinaryResolver resolves a known SDK tool inside a toolchain directory.
type ToolBinaryResolver func(toolchainDir, command string) (string, error)

// PlatformBinaryNamer applies platform executable naming, e.g. .exe on Windows.
type PlatformBinaryNamer func(string) string

// ResolveToolPath resolves command inside tcDir, first via the known-tool
// resolver, then by scanning bin/ and tools/bin/. When false, the caller may
// still resolve through PATH.
func ResolveToolPath(tcDir, command string, known ToolBinaryResolver, platformBinary PlatformBinaryNamer) (string, bool) {
	if known != nil {
		if toolPath, err := known(tcDir, command); err == nil {
			return toolPath, true
		}
	}
	if platformBinary == nil {
		platformBinary = func(name string) string { return name }
	}
	binaryName := platformBinary(command)
	for _, subDir := range []string{"bin", filepath.Join("tools", "bin")} {
		candidate := filepath.Join(tcDir, subDir, binaryName)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, true
		}
	}
	return command, false
}

// LookPathInEnv resolves a bare command name against the PATH carried in
// environ, honoring PATHEXT on Windows.
func LookPathInEnv(command string, environ []string) (string, bool) {
	if strings.ContainsRune(command, '/') || strings.ContainsRune(command, filepath.Separator) {
		return command, false
	}
	pathVal, _ := LookupValue(environ, "PATH")
	pathext, _ := LookupValue(environ, "PATHEXT")
	exts := executableExtensions(command, pathext)
	for _, dir := range filepath.SplitList(pathVal) {
		if dir == "" {
			continue
		}
		base := filepath.Join(dir, command)
		for _, ext := range exts {
			candidate := base + ext
			if isRegularExecutable(candidate) {
				return candidate, true
			}
		}
	}
	return command, false
}

func executableExtensions(command, pathext string) []string {
	if runtime.GOOS != "windows" || filepath.Ext(command) != "" {
		return []string{""}
	}
	if pathext == "" {
		pathext = ".COM;.EXE;.BAT;.CMD"
	}
	var exts []string
	for p := range strings.SplitSeq(pathext, ";") {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if !strings.HasPrefix(p, ".") {
			p = "." + p
		}
		exts = append(exts, strings.ToLower(p))
	}
	return exts
}

func isRegularExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		return false
	}
	if runtime.GOOS == "windows" {
		return true
	}
	return info.Mode()&0o111 != 0
}
