//go:build !windows

package lifecycle_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const shellPathMarker = "# cjv (managed by cjv, do not edit)"

// fakeShellConfig points HOME at a temporary .bashrc so the tests observe
// the PATH block a first install writes. Windows writes the registry
// instead; that policy is covered by reachable's guarded tests.
func fakeShellConfig(t *testing.T) string {
	t.Helper()
	userHome := t.TempDir()
	rc := filepath.Join(userHome, ".bashrc")
	require.NoError(t, os.WriteFile(rc, []byte("# existing\n"), 0o644))
	t.Setenv("HOME", userHome)
	t.Setenv(config.EnvNoPathSetup, "")
	return rc
}

func TestFirstDefaultInstallConfiguresPathOnlyWhenRequested(t *testing.T) {
	for _, configure := range []bool{true, false} {
		t.Run(map[bool]string{true: "configure", false: "leave"}[configure], func(t *testing.T) {
			home, _, _ := prepareInstallTest(t)
			rc := fakeShellConfig(t)
			opts := lifecycle.Options{ConfigurePath: configure}

			require.NoError(t, lifecycle.Install(t.Context(), lifecycle.InstallRequest{Toolchain: "lts"}, opts))
			data, err := os.ReadFile(rc)
			require.NoError(t, err)
			if configure {
				assert.Equal(t, 1, strings.Count(string(data), shellPathMarker))
				assert.Contains(t, string(data), filepath.Join(home, "bin"))
			} else {
				assert.Equal(t, "# existing\n", string(data))
			}

			// A later install does not publish a first default, so it leaves
			// PATH alone even when the policy allows configuring it.
			require.NoError(t, os.WriteFile(rc, []byte("# existing\n"), 0o644))
			require.NoError(t, lifecycle.Install(t.Context(), lifecycle.InstallRequest{Toolchain: "sts"}, opts))
			data, err = os.ReadFile(rc)
			require.NoError(t, err)
			assert.Equal(t, "# existing\n", string(data))
		})
	}
}
