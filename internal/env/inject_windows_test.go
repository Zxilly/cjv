//go:build windows

package env

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildProxyEnvCollapsesWindowsEnvKeyCasing(t *testing.T) {
	baseEnv := []string{
		"Path=C:\\Windows",
		"path=C:\\Tools",
		"cjv_toolchain=old",
	}
	cfg := &EnvConfig{
		Vars: map[string]string{},
		PathPrepend: PathPrepend{
			Entries: []string{"C:\\SDK\\tools\\bin"},
		},
	}

	result := BuildProxyEnv(baseEnv, ProxyEnvContext{
		Cfg: cfg, CjvBinDir: `C:\Users\user\.cjv\bin`, ToolchainBinDir: `C:\SDK\bin`,
		Recursion: 1, ToolchainName: "lts-1.0.5",
	})

	assert.Equal(t, 1, countEnvKeys(result, "PATH"))
	assert.Equal(t, 1, countEnvKeys(result, "CJV_TOOLCHAIN"))
	assert.Equal(t, 1, countEnvKeys(result, "CJV_RECURSION_COUNT"))

	pathParts := strings.Split(findEnvValueFold(result, "PATH"), ";")
	assert.Equal(t, `C:\Users\user\.cjv\bin`, pathParts[0])
	assert.Equal(t, `C:\SDK\bin`, pathParts[len(pathParts)-1])
	assert.Contains(t, pathParts, `C:\SDK\tools\bin`)
	assert.Equal(t, "lts-1.0.5", findEnvValueFold(result, "CJV_TOOLCHAIN"))
}

func countEnvKeys(env []string, key string) int {
	count := 0
	for _, entry := range env {
		k, _, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(k, key) {
			count++
		}
	}
	return count
}

func findEnvValueFold(env []string, key string) string {
	for _, entry := range env {
		k, v, ok := strings.Cut(entry, "=")
		if ok && strings.EqualFold(k, key) {
			return v
		}
	}
	return ""
}
