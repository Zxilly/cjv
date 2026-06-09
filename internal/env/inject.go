package env

import (
	"os"
	"runtime"
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
		if strings.EqualFold(k, "PATH") {
			pathKey = canonicalEnvKey(k)
		}
	}

	// SDK env vars intentionally override user values to match the active toolchain.
	for k, v := range ctx.Cfg.Vars {
		setEnv(k, v)
	}

	// PATH construction: CjvBinDir first so nested tool invocations still hit
	// the proxy, then SDK entries, then existing PATH without duplicates.
	// On Windows, ToolchainBinDir is appended so executables under tools/bin
	// can resolve DLLs in the SDK's bin directory without shadowing PATH.
	{
		var entries []string
		if path, ok := envMap[pathKey]; ok {
			entries = strings.Split(path, string(os.PathListSeparator))
		} else {
			displayKeys[pathKey] = "PATH"
			order = append(order, pathKey)
		}

		existingSet := make(map[string]bool, len(entries)+len(ctx.Cfg.PathPrepend)+len(ctx.Cfg.PathAppend)+2)
		all := make([]string, 0, len(ctx.Cfg.PathPrepend)+len(entries)+len(ctx.Cfg.PathAppend)+2)
		appendUnique := func(entry string) {
			if entry == "" {
				return
			}
			key := canonicalEnvKey(entry)
			if existingSet[key] {
				return
			}
			existingSet[key] = true
			all = append(all, entry)
		}

		appendUnique(ctx.CjvBinDir)
		for _, e := range ctx.Cfg.PathPrepend {
			appendUnique(e)
		}
		for _, e := range entries {
			appendUnique(e)
		}
		for _, e := range ctx.Cfg.PathAppend {
			appendUnique(e)
		}
		if runtime.GOOS == "windows" {
			appendUnique(ctx.ToolchainBinDir)
		}
		envMap[pathKey] = strings.Join(all, string(os.PathListSeparator))
	}

	setEnv(config.EnvRecursionCount, strconv.Itoa(ctx.Recursion+1))

	if ctx.ToolchainName != "" {
		setEnv(config.EnvToolchain, ctx.ToolchainName)
	}

	// order is already deduplicated by setEnv
	result := make([]string, 0, len(order)+len(hiddenEntries))
	for _, k := range order {
		result = append(result, displayKeys[k]+"="+envMap[k])
	}
	result = append(result, hiddenEntries...)
	return result
}

// BuildToolchainEnv constructs the environment for an interactive shell
// session that wants the SDK/runtime paths directly available. Unlike
// BuildProxyEnv it does not add CJV_HOME/bin and does not set proxy-internal
// CJV_* variables.
func BuildToolchainEnv(baseEnv []string, cfg *EnvConfig) []string {
	if cfg == nil {
		cfg = NewEnvConfig()
	}

	envMap := make(map[string]string)
	displayKeys := make(map[string]string)
	pathKey := canonicalEnvKey("PATH")
	var order []string
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
			hiddenEntries = append(hiddenEntries, e)
			continue
		}
		setEnv(k, v)
		if strings.EqualFold(k, "PATH") {
			pathKey = canonicalEnvKey(k)
		}
	}

	for k, v := range cfg.Vars {
		setEnv(k, v)
	}

	var entries []string
	if path, ok := envMap[pathKey]; ok {
		entries = strings.Split(path, string(os.PathListSeparator))
	} else {
		displayKeys[pathKey] = "PATH"
		order = append(order, pathKey)
	}

	existingSet := make(map[string]bool, len(entries)+len(cfg.PathPrepend)+len(cfg.PathAppend))
	all := make([]string, 0, len(cfg.PathPrepend)+len(entries)+len(cfg.PathAppend))
	appendUnique := func(entry string) {
		if entry == "" {
			return
		}
		key := canonicalEnvKey(entry)
		if existingSet[key] {
			return
		}
		existingSet[key] = true
		all = append(all, entry)
	}
	for _, e := range cfg.PathPrepend {
		appendUnique(e)
	}
	for _, e := range entries {
		appendUnique(e)
	}
	for _, e := range cfg.PathAppend {
		appendUnique(e)
	}
	envMap[pathKey] = strings.Join(all, string(os.PathListSeparator))

	result := make([]string, 0, len(order)+len(hiddenEntries))
	for _, k := range order {
		result = append(result, displayKeys[k]+"="+envMap[k])
	}
	result = append(result, hiddenEntries...)
	return result
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
