package settingscmd

import (
	"encoding/json"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/cli/clitest"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func decodeSettingsJSON(t *testing.T, args ...string) map[string]any {
	t.Helper()
	out, err := clitest.Execute(t, clitest.Root(Register), append(args, "--json")...)
	require.NoError(t, err)
	decoder := json.NewDecoder(strings.NewReader(out))
	var result map[string]any
	require.NoError(t, decoder.Decode(&result), out)
	require.ErrorIs(t, decoder.Decode(new(any)), io.EOF, "success is exactly one JSON document")
	return result
}

func TestSettingJSONReportsChangesAndNoOp(t *testing.T) {
	configtest.Isolate(t, t.TempDir())
	for _, changed := range []bool{true, false} {
		result := decodeSettingsJSON(t, "set", "auto-install", "false")
		assert.Equal(t, "auto-install", result["key"])
		assert.Equal(t, false, result["value"])
		assert.Equal(t, changed, result["changed"])
	}
	result := decodeSettingsJSON(t, "set", "auto-self-update", "disable")
	assert.Equal(t, "disable", result["value"])
	result = decodeSettingsJSON(t, "set", "home", "")
	assert.Equal(t, "", result["value"], "clearing a setting returns its value, not a display placeholder")
}

func TestOverrideJSONLifecycle(t *testing.T) {
	home, project := t.TempDir(), t.TempDir()
	configtest.Isolate(t, home)
	t.Chdir(project)
	result := decodeSettingsJSON(t, "override", "list")
	assert.Equal(t, []any{}, result["overrides"])
	result = decodeSettingsJSON(t, "override", "set", "lts-1.0.5")
	assert.Equal(t, config.NormalizePath(project), result["directory"])
	assert.Equal(t, "lts-1.0.5", result["toolchain"])
	result = decodeSettingsJSON(t, "override", "list")
	assert.Equal(t, []any{map[string]any{"directory": config.NormalizePath(project), "toolchain": "lts-1.0.5"}}, result["overrides"])
	_, err := clitest.Execute(t, clitest.Root(Register), "override", "unset", "typo")
	require.Error(t, err)
	_, settings, err := config.LoadDefaultSettings()
	require.NoError(t, err)
	assert.Contains(t, settings.Overrides, config.NormalizePath(project))
	result = decodeSettingsJSON(t, "override", "unset")
	assert.Equal(t, []any{config.NormalizePath(project)}, result["removed"])
	result = decodeSettingsJSON(t, "override", "unset", "--nonexistent")
	assert.Equal(t, []any{}, result["removed"])
	missing := config.NormalizePath(filepath.Join(home, "missing-project"))
	decodeSettingsJSON(t, "override", "set", "nightly", "--path", missing)
	result = decodeSettingsJSON(t, "override", "unset", "--nonexistent")
	assert.Equal(t, []any{missing}, result["removed"])
}
