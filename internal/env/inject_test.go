package env

import (
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var sep = string(os.PathListSeparator)

func TestBuildProxyEnv(t *testing.T) {
	baseEnv := []string{
		"PATH=/usr/bin" + sep + "/home/user/.cjv/bin",
		"HOME=/home/user",
	}

	cfg := &EnvConfig{
		Vars: map[string]string{"CANGJIE_HOME": "/sdk"},
		PathPrepend: PathPrepend{
			Entries: []string{"/sdk/bin", "/sdk/tools/bin"},
		},
	}

	result := BuildProxyEnv(baseEnv, ProxyEnvContext{
		Cfg: cfg, CjvBinDir: "/home/user/.cjv/bin", ToolchainBinDir: "/sdk/bin",
	})

	// CANGJIE_HOME should be set
	assertEnvContains(t, result, "CANGJIE_HOME=/sdk")

	// PATH should keep .cjv/bin first so nested invocations still hit the proxy.
	pathVal := findEnvValue(result, "PATH")
	parts := strings.Split(pathVal, sep)
	assert.Equal(t, "/home/user/.cjv/bin", parts[0])
	assert.Contains(t, pathVal, "/sdk/bin")

	// CJV_RECURSION_COUNT should be 1
	assertEnvContains(t, result, "CJV_RECURSION_COUNT=1")
}

func TestBuildProxyEnvPreservesExisting(t *testing.T) {
	baseEnv := []string{
		"PATH=/usr/bin",
		"HOME=/home/user",
		"CUSTOM_VAR=hello",
	}

	cfg := &EnvConfig{
		Vars:        map[string]string{"NEW_VAR": "world"},
		PathPrepend: PathPrepend{},
	}

	result := BuildProxyEnv(baseEnv, ProxyEnvContext{
		Cfg: cfg, CjvBinDir: "/home/user/.cjv/bin", ToolchainBinDir: "/sdk/bin",
	})

	assertEnvContains(t, result, "HOME=/home/user")
	assertEnvContains(t, result, "CUSTOM_VAR=hello")
	assertEnvContains(t, result, "NEW_VAR=world")
}

func TestBuildProxyEnvIncrementsRecursion(t *testing.T) {
	baseEnv := []string{"PATH=/usr/bin"}
	cfg := &EnvConfig{Vars: make(map[string]string)}

	result := BuildProxyEnv(baseEnv, ProxyEnvContext{Cfg: cfg, Recursion: 5})
	assertEnvContains(t, result, "CJV_RECURSION_COUNT=6")
}

func assertEnvContains(t *testing.T, env []string, expected string) {
	t.Helper()
	if slices.Contains(env, expected) {
		return
	}
	t.Errorf("env does not contain %q, got: %v", expected, env)
}

func findEnvValue(env []string, key string) string {
	for _, e := range env {
		k, v, _ := strings.Cut(e, "=")
		if k == key {
			return v
		}
	}
	return ""
}

func TestBuildProxyEnv_EmptyBaseEnv(t *testing.T) {
	// When baseEnv is empty (no PATH set), the function should still
	// construct a valid environment with the required entries.
	cfg := NewEnvConfig()
	cfg.PathPrepend.Entries = []string{"/sdk/bin"}

	result := BuildProxyEnv(nil, ProxyEnvContext{
		Cfg: cfg, CjvBinDir: "/cjv/bin", ToolchainBinDir: "/tc/bin", ToolchainName: "lts-1.0.5",
	})

	// Should have PATH with cjv bin and SDK entries
	var pathValue string
	for _, e := range result {
		if strings.HasPrefix(e, "PATH=") || strings.HasPrefix(e, "Path=") {
			pathValue = e[5:] // skip "PATH=" or "Path="
			break
		}
	}
	assert.Contains(t, pathValue, "/cjv/bin")
	assert.Contains(t, pathValue, "/sdk/bin")
}

func TestBuildProxyEnv_EnvTomlOverridesBase(t *testing.T) {
	// SDK-specific vars from env.toml should override the base env.
	cfg := NewEnvConfig()
	cfg.Vars["CANGJIE_HOME"] = "/new/sdk"

	baseEnv := []string{"CANGJIE_HOME=/old/sdk", "OTHER=keep"}
	result := BuildProxyEnv(baseEnv, ProxyEnvContext{
		Cfg: cfg, CjvBinDir: "/cjv/bin", ToolchainBinDir: "/tc/bin",
	})

	var cjHome string
	for _, e := range result {
		if strings.HasPrefix(e, "CANGJIE_HOME=") {
			cjHome = e[len("CANGJIE_HOME="):]
		}
	}
	assert.Equal(t, "/new/sdk", cjHome,
		"env.toml var should override base env var")
}

func TestBuildProxyEnv_PathDeduplication(t *testing.T) {
	// If /cjv/bin is already in the base PATH, it should not appear twice.
	sep := string(os.PathListSeparator)
	cfg := NewEnvConfig()
	baseEnv := []string{"PATH=/cjv/bin" + sep + "/usr/bin"}

	result := BuildProxyEnv(baseEnv, ProxyEnvContext{
		Cfg: cfg, CjvBinDir: "/cjv/bin", ToolchainBinDir: "/tc/bin",
	})

	var pathValue string
	for _, e := range result {
		if strings.HasPrefix(e, "PATH=") || strings.HasPrefix(e, "Path=") {
			pathValue = strings.SplitN(e, "=", 2)[1]
			break
		}
	}
	count := 0
	for _, p := range strings.Split(pathValue, sep) {
		if p == "/cjv/bin" {
			count++
		}
	}
	assert.Equal(t, 1, count, "PATH should not contain duplicates")
}

func TestBuildProxyEnv_SetsToolchainName(t *testing.T) {
	cfg := NewEnvConfig()
	result := BuildProxyEnv(nil, ProxyEnvContext{
		Cfg: cfg, CjvBinDir: "/cjv/bin", ToolchainBinDir: "/tc/bin", ToolchainName: "lts-1.0.5",
	})

	var found bool
	for _, e := range result {
		if e == "CJV_TOOLCHAIN=lts-1.0.5" {
			found = true
			break
		}
	}
	assert.True(t, found, "CJV_TOOLCHAIN should be set")
}
