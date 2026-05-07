package component

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplyEnvAddsStdxPathsOnlyWhenInstalled(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)
	tcName := "lts-1.0.5"
	tcDir := filepath.Join(home, "toolchains", tcName)
	require.NoError(t, os.MkdirAll(tcDir, 0o755))

	vars := map[string]string{}
	ApplyEnv(vars, tcDir)
	assert.Empty(t, vars)

	require.NoError(t, WriteManifest(tcDir, Stdx, []string{"dynamic/libfoo.so"}))
	ApplyEnv(vars, tcDir)

	stdxRoot, err := config.StdxDirFor(tcName)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(stdxRoot, "dynamic"), vars[EnvStdxDynamic])
	assert.Equal(t, filepath.Join(stdxRoot, "static"), vars[EnvStdxStatic])
}
