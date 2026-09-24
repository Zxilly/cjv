//go:build !windows

package reachable

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// shellRCFiles are the POSIX shell configs ConfigurePath edits, plus fish.
var shellRCFiles = []string{".profile", ".bashrc", ".zshrc", ".zprofile", ".config/fish/config.fish"}

// fakeUserHome points the user's home at a temporary directory holding every
// shell config ConfigurePath edits, so the tests observe real PATH writes
// without touching the developer's files. CJV_HOME is isolated separately.
func fakeUserHome(t *testing.T) (userHome, binDir string) {
	t.Helper()
	cjvHome := t.TempDir()
	config.IsolateForTest(t, cjvHome)
	userHome = t.TempDir()
	t.Setenv("HOME", userHome)
	t.Setenv(config.EnvNoPathSetup, "")
	for _, rc := range shellRCFiles {
		path := filepath.Join(userHome, rc)
		require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
		require.NoError(t, os.WriteFile(path, []byte("# existing\n"), 0o644))
	}
	return userHome, filepath.Join(cjvHome, "bin")
}

func readRC(t *testing.T, userHome, rc string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(userHome, rc))
	require.NoError(t, err)
	return string(data)
}

func TestConfigurePathWritesEveryShellConfigOnce(t *testing.T) {
	userHome, binDir := fakeUserHome(t)

	ConfigurePath()
	ConfigurePath()

	for _, rc := range shellRCFiles {
		content := readRC(t, userHome, rc)
		assert.Equal(t, 1, strings.Count(content, markerStart), rc)
		assert.Contains(t, content, binDir, rc)
		assert.Contains(t, content, "# existing", rc)
		if strings.HasSuffix(rc, "config.fish") {
			assert.Contains(t, content, "fish_add_path", rc)
		} else {
			assert.Contains(t, content, "export PATH=", rc)
		}
	}
}

func TestConfigurePathSkipsWhenDisabledByEnv(t *testing.T) {
	userHome, _ := fakeUserHome(t)
	t.Setenv(config.EnvNoPathSetup, "1")

	ConfigurePath()

	for _, rc := range shellRCFiles {
		assert.Equal(t, "# existing\n", readRC(t, userHome, rc), rc)
	}
}

func TestRemovePathUndoesConfigurePath(t *testing.T) {
	userHome, _ := fakeUserHome(t)

	ConfigurePath()
	RemovePath()

	for _, rc := range shellRCFiles {
		assert.Equal(t, "# existing\n", readRC(t, userHome, rc), rc)
	}
}

func TestEnsureConfiguresPathOnlyWhenPolicyAsks(t *testing.T) {
	userHome, binDir := fakeUserHome(t)

	require.NoError(t, Ensure(Policy{}))
	for _, rc := range shellRCFiles {
		assert.NotContains(t, readRC(t, userHome, rc), markerStart, rc)
	}

	require.NoError(t, Ensure(Policy{ConfigurePath: true}))
	for _, rc := range shellRCFiles {
		assert.Contains(t, readRC(t, userHome, rc), binDir, rc)
	}
}
