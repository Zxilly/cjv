//go:build darwin

package env_test

import (
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/env"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuntimePreservesExplicitSDKRoot(t *testing.T) {
	config.IsolateForTest(t, t.TempDir())
	for _, constructionRoot := range []string{"/construction-sdk", ""} {
		t.Setenv("SDKROOT", constructionRoot)
		rt, err := env.RuntimeForToolchain(t.TempDir(), "lts-1.0.5")
		require.NoError(t, err)
		base := []string{"SDKROOT=/caller-sdk"}
		for _, got := range [][]string{rt.ProxyEnv(base, 0), rt.ToolchainEnv(base)} {
			value, _ := env.LookupValue(got, "SDKROOT")
			assert.Equal(t, "/caller-sdk", value)
		}
		assert.NotContains(t, rt.Contributions(base).Vars, "SDKROOT")
	}
}
