package env

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadToolchainEnvAppliesComponentEnv(t *testing.T) {
	tcDir := t.TempDir()

	got := LoadToolchainEnv(tcDir, func(vars map[string]string, _ string) {
		vars["COMPONENT"] = "enabled"
	})

	assert.Equal(t, tcDir, got.Vars["CANGJIE_HOME"])
	assert.Equal(t, "enabled", got.Vars["COMPONENT"])
}

func TestLoadToolchainEnv_DerivesCangjieHome(t *testing.T) {
	tcDir := t.TempDir()

	cfg := LoadToolchainEnv(tcDir, nil)

	assert.Equal(t, tcDir, cfg.Vars["CANGJIE_HOME"])
}
