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
	"github.com/Zxilly/cjv/internal/sdktools"
)

// Runtime is the deep module for Cangjie runtime environment assembly. It
// carries the active toolchain plus the derived SDK environment and exposes
// narrow views for proxy children, direct toolchain execution, and shell output.
type Runtime struct {
	Active    resolve.ActiveToolchain
	CjvBinDir string
	cfg       *EnvConfig
}

// Contributions describes the SDK changes without including inherited
// variables or paths. Each view owns its maps and slices so callers cannot
// change subsequent environments built from the Runtime.
type Contributions struct {
	Vars               map[string]string
	PathPrepend        []string
	PathAppend         []string
	LibraryPathKey     string
	LibraryPathPrepend []string
}

func ResolveRuntime(ctx context.Context, tcOverride string) (Runtime, error) {
	active, err := resolve.Active(ctx, tcOverride)
	if err != nil {
		return Runtime{}, err
	}
	return runtimeForActive(active)
}

func ResolveTargetRuntime(ctx context.Context, tcOverride, target string) (Runtime, error) {
	active, err := resolve.ActiveTarget(ctx, tcOverride, target)
	if err != nil {
		return Runtime{}, err
	}
	return runtimeForActive(active)
}

func RuntimeForToolchain(dir, name string) (Runtime, error) {
	return runtimeForActive(resolve.ActiveToolchain{Dir: dir, Name: name})
}

func runtimeForActive(active resolve.ActiveToolchain) (Runtime, error) {
	cfg := LoadToolchainEnv(active.Dir)
	binDir, err := config.BinDir()
	if err != nil {
		return Runtime{}, fmt.Errorf("failed to determine bin directory: %w", err)
	}
	return Runtime{
		Active:    active,
		CjvBinDir: binDir,
		cfg:       cfg,
	}, nil
}

func (r Runtime) ProxyEnv(baseEnv []string, recursion int) []string {
	return BuildProxyEnv(baseEnv, ProxyEnvContext{
		Cfg:             r.cfg,
		CjvBinDir:       r.CjvBinDir,
		ToolchainBinDir: filepath.Join(r.Active.Dir, "bin"),
		Recursion:       recursion,
		ToolchainName:   r.Active.Name,
	})
}

func (r Runtime) ToolchainEnv(baseEnv []string) []string {
	return BuildToolchainEnv(baseEnv, r.cfg)
}

// Contributions reports the SDK ingredients and platform defaults needed for
// baseEnv. It does not include proxy-internal variables or merge inherited paths.
func (r Runtime) Contributions(baseEnv []string) Contributions {
	return environmentContributions(r.cfg, baseEnv)
}

// ShellScript formats only the changes needed to prepare baseEnv for direct
// SDK use. Proxy selection and recursion state are left out of shell setup.
func (r Runtime) ShellScript(baseEnv []string, shell ShellType) string {
	return FormatEnvDiff(ComputeEnvDiff(baseEnv, r.ToolchainEnv(baseEnv)), shell)
}

// ResolveToolPath resolves command inside tcDir, first through the known SDK
// tool layout, then by scanning bin/ and tools/bin/ for the platform binary
// name. When false, the caller may still resolve through PATH.
func ResolveToolPath(tcDir, command string) (string, bool) {
	if toolPath, err := sdktools.ResolveInstalledToolBinary(tcDir, command); err == nil {
		return toolPath, true
	}
	binaryName := sdktools.PlatformBinaryName(command)
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
