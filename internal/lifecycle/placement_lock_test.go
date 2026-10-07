package lifecycle

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlacementDoesNotBlockRecoveryDuringPreparation(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	preparation, err := dist.BeginPreparation(t.Context(), filepath.Join(home, "downloads"))
	require.NoError(t, err)
	defer preparation.Close()
	recoverHome := func() {
		ctx, cancel := context.WithTimeout(t.Context(), time.Second)
		defer cancel()
		require.NoError(t, toolchain.RecoverHomeContext(ctx))
	}
	acq := acquisition{
		fetch: func(context.Context, *dist.Preparation) (string, error) {
			recoverHome()
			return "unused", nil
		},
		extract: func(_ context.Context, _, prepared string) error {
			compiler := filepath.Join(prepared, "bin", sdktools.PlatformBinaryName("cjc"))
			require.NoError(t, os.MkdirAll(filepath.Dir(compiler), 0o755))
			require.NoError(t, os.WriteFile(compiler, []byte("compiler"), 0o755))
			// Private preparation can coexist with ordinary proxy recovery.
			recoverHome()
			require.FileExists(t, compiler)
			return nil
		},
	}
	require.NoError(t, placeToolchain(t.Context(), "background-sdk", false, "", preparation, acq, nil, Options{}))
	assert.FileExists(t, filepath.Join(home, "toolchains", "background-sdk", "bin", sdktools.PlatformBinaryName("cjc")))
}

func TestPlacementRechecksDestinationAfterPreparation(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	preparation, err := dist.BeginPreparation(t.Context(), filepath.Join(home, "downloads"))
	require.NoError(t, err)
	defer preparation.Close()
	const name = "shared-sdk"
	dest := filepath.Join(home, "toolchains", name)
	acq := acquisition{
		fetch: func(context.Context, *dist.Preparation) (string, error) { return "unused", nil },
		extract: func(_ context.Context, _, prepared string) error {
			compiler := filepath.Join(prepared, "bin", sdktools.PlatformBinaryName("cjc"))
			require.NoError(t, os.MkdirAll(filepath.Dir(compiler), 0o755))
			require.NoError(t, os.WriteFile(compiler, []byte("new compiler"), 0o755))
			// Another home mutation may publish this name while preparation is
			// unlocked. Simulate it through the real directory-link entry point.
			source := t.TempDir()
			sourceCompiler := filepath.Join(source, "bin", sdktools.PlatformBinaryName("cjc"))
			require.NoError(t, os.MkdirAll(filepath.Dir(sourceCompiler), 0o755))
			require.NoError(t, os.WriteFile(sourceCompiler, []byte("linked compiler"), 0o755))
			require.NoError(t, LinkToolchainDir(name, source))
			return nil
		},
	}
	err = placeToolchain(t.Context(), name, false, "", preparation, acq, nil, Options{})
	var already *cjverr.ToolchainAlreadyInstalledError
	require.ErrorAs(t, err, &already)
	content, err := os.ReadFile(filepath.Join(dest, "bin", sdktools.PlatformBinaryName("cjc")))
	require.NoError(t, err)
	assert.Equal(t, "linked compiler", string(content))
	assert.NoDirExists(t, config.StagingDir(dest))
}
