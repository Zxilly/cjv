package toolchaincmd

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/cli/clitest"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for install's cobra handler, which turns flags into an
// InstallRequest. Resolution, download, extraction, validation and the
// transactional swap are lifecycle behaviour, tested in internal/lifecycle.

func TestRunInstallInstallsNamedToolchain(t *testing.T) {
	validManifestHome(t)
	require.NoError(t, newInstall(newTestContext()).run(&cobra.Command{}, []string{"lts"}))
	installed, err := toolchain.ListInstalled()
	require.NoError(t, err)
	assert.Equal(t, []string{"lts"}, installed)
}

func TestRunInstallRejectsEmptyName(t *testing.T) {
	configtest.Isolate(t, t.TempDir())
	err := newInstall(newTestContext()).run(&cobra.Command{}, []string{""})
	assert.Error(t, err, "empty name should fail")
}

func TestInstallWithComponentsEmitsSingleJSON(t *testing.T) {
	home, name := clitest.NightlyDistHome(t)
	stdout, err := execute(t, "--json", "install", "nightly", "--component", "docs")
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(home, "docs", name, "main", "index.html"))
	var result installResult
	require.NoError(t, json.Unmarshal([]byte(stdout), &result), stdout)
	require.Equal(t, []string{"docs"}, result.Components)

	// Idempotent SDK installation has the same success semantics in JSON mode.
	stdout, err = execute(t, "--json", "install", "nightly")
	require.NoError(t, err)
	require.True(t, json.Valid([]byte(stdout)), stdout)
}
