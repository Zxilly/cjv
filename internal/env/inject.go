package env

import (
	"maps"
	"os"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/Zxilly/cjv/internal/config"
)

// ProxyEnvContext groups the toolchain-related parameters for BuildProxyEnv.
type ProxyEnvContext struct {
	Cfg             *EnvConfig
	CjvBinDir       string
	ToolchainBinDir string
	Recursion       int
	ToolchainName   string
}

// BuildProxyEnv constructs the environment for a proxy subprocess.
// It applies the toolchain's vars, keeps CjvBinDir at the front of PATH so
// nested invocations still route through the proxy, and increments the
// recursion counter. On Windows, ToolchainBinDir is appended so tools under
// tools/bin can still load DLLs shipped in the SDK bin directory without
// taking precedence over PATH tools.
func BuildProxyEnv(baseEnv []string, ctx ProxyEnvContext) []string {
	return buildEnvironment(baseEnv, ctx.Cfg, &ctx)
}

// BuildToolchainEnv constructs the environment for an interactive shell
// session that wants the SDK/runtime paths directly available. Unlike
// BuildProxyEnv it does not add CJV_HOME/bin and does not set proxy-internal
// CJV_* variables.
func BuildToolchainEnv(baseEnv []string, cfg *EnvConfig) []string {
	return buildEnvironment(baseEnv, cfg, nil)
}

func buildEnvironment(baseEnv []string, cfg *EnvConfig, proxy *ProxyEnvContext) []string {
	contributed := environmentContributions(cfg, baseEnv)
	envMap := make(map[string]string)
	displayKeys := make(map[string]string)
	pathKey := canonicalEnvKey("PATH")
	var order []string
	// Windows uses hidden env vars like "=C:=C:\path" to track per-drive
	// current directories. They have no normal key so we pass them through
	// verbatim to the child process.
	var hiddenEntries []string

	setEnv := func(key, value string) {
		canonical := canonicalEnvKey(key)
		if _, exists := displayKeys[canonical]; !exists {
			displayKeys[canonical] = key
			order = append(order, canonical)
		}
		envMap[canonical] = value
	}

	for _, e := range baseEnv {
		k, v, ok := strings.Cut(e, "=")
		if !ok {
			continue
		}
		if k == "" {
			// Preserve Windows hidden entries (e.g. "=C:=C:\Users\user")
			hiddenEntries = append(hiddenEntries, e)
			continue
		}
		setEnv(k, v)
	}

	// SDK env vars intentionally override user values to match the active toolchain.
	for _, k := range slices.Sorted(maps.Keys(contributed.Vars)) {
		setEnv(k, contributed.Vars[k])
	}

	// Proxy children must find cjv before SDK tools; a directly prepared shell
	// starts with the SDK directories. Both otherwise share the same merge.
	var proxyPrepend, proxyAppend []string
	if proxy != nil {
		proxyPrepend = []string{proxy.CjvBinDir}
		if runtime.GOOS == "windows" {
			proxyAppend = []string{proxy.ToolchainBinDir}
		}
	}
	setEnv("PATH", mergePathLists(proxyPrepend, contributed.PathPrepend,
		strings.Split(envMap[pathKey], string(os.PathListSeparator)), contributed.PathAppend, proxyAppend))

	if key := contributed.LibraryPathKey; key != "" && len(contributed.LibraryPathPrepend) > 0 {
		setEnv(key, mergePathLists(contributed.LibraryPathPrepend,
			strings.Split(envMap[canonicalEnvKey(key)], string(os.PathListSeparator))))
	}
	if proxy != nil {
		setEnv(config.EnvRecursionCount, strconv.Itoa(proxy.Recursion+1))
		if proxy.ToolchainName != "" {
			setEnv(config.EnvToolchain, proxy.ToolchainName)
		}
	}

	// order is already deduplicated by setEnv
	result := make([]string, 0, len(order)+len(hiddenEntries))
	for _, k := range order {
		result = append(result, displayKeys[k]+"="+envMap[k])
	}
	result = append(result, hiddenEntries...)
	return result
}

func mergePathLists(groups ...[]string) string {
	seen := make(map[string]bool)
	var merged []string
	for _, group := range groups {
		for _, entry := range group {
			key := canonicalEnvKey(entry)
			if entry == "" || seen[key] {
				continue
			}
			seen[key] = true
			merged = append(merged, entry)
		}
	}
	return strings.Join(merged, string(os.PathListSeparator))
}

func canonicalEnvKey(key string) string {
	if runtime.GOOS == "windows" {
		return strings.ToUpper(key)
	}
	return key
}

// LookupValue returns the value of key from a KEY=VALUE environment slice,
// matching keys case-insensitively on Windows (mirroring the OS). The last
// matching entry wins, as the OS resolves duplicate variables. This is the
// single source of the env-key matching rule, shared with the env builders
// above so a custom-env lookup (e.g. cjv run's PATH resolution) cannot drift.
func LookupValue(environ []string, key string) (string, bool) {
	want := canonicalEnvKey(key)
	value, found := "", false
	for _, kv := range environ {
		k, v, ok := strings.Cut(kv, "=")
		if !ok {
			continue
		}
		if canonicalEnvKey(k) == want {
			value, found = v, true
		}
	}
	return value, found
}
