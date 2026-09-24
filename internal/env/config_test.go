package env

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadToolchainEnvAppliesComponentEnv(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	tcDir := filepath.Join(home, "toolchains", "lts-1.0.5")
	require.NoError(t, os.MkdirAll(tcDir, 0o755))
	require.NoError(t, component.WriteManifest(tcDir, component.Stdx, []string{"dynamic/libfoo.so"}))
	stdxRoot, err := config.StdxDirFor("lts-1.0.5")
	require.NoError(t, err)

	got := LoadToolchainEnv(tcDir)

	assert.Equal(t, tcDir, got.Vars["CANGJIE_HOME"])
	assert.Equal(t, filepath.Join(stdxRoot, "dynamic"), got.Vars[component.EnvStdxDynamic])
}

func TestLoadToolchainEnv_DerivesCangjieHome(t *testing.T) {
	tcDir := t.TempDir()

	cfg := LoadToolchainEnv(tcDir)

	assert.Equal(t, tcDir, cfg.Vars["CANGJIE_HOME"])
}
