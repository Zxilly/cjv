package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/output"
	componentlib "github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupFakeToolchainForCLI(t *testing.T, home, name string) string {
	t.Helper()
	tcDir := filepath.Join(home, "toolchains", name)
	binDir := filepath.Join(tcDir, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tcDir, "tools", "bin"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tcDir, "tools", "lib"), 0o755))
	require.NoError(t, os.MkdirAll(filepath.Join(tcDir, "runtime", "lib", hostBackendFixture()), 0o755))
	cjcName := "cjc"
	if runtime.GOOS == "windows" {
		cjcName = "cjc.exe"
	}
	require.NoError(t, os.WriteFile(filepath.Join(binDir, cjcName), []byte("stub"), 0o755))
	return tcDir
}

func hostBackendFixture() string {
	osName := runtime.GOOS
	if osName == "darwin" {
		osName = "mac"
	}
	arch := runtime.GOARCH
	switch arch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "aarch64"
	}
	return osName + "_" + arch + "_cjnative"
}

// executeEnvsetup runs the envsetup command through a real cobra command tree
// so flag parsing (the global --json persistent flag, the local --shell flag,
// and the +toolchain positional argument) is exercised exactly as in
// production rather than bypassed.
func executeEnvsetup(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var jsonOn bool
	root := &cobra.Command{
		Use:           "cjv",
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			output.SetJSONMode(jsonOn)
		},
	}
	root.PersistentFlags().BoolVar(&jsonOn, "json", false, "")
	root.AddCommand(newEnvsetupCmd())
	t.Cleanup(func() { output.SetJSONMode(false) })

	var buf bytes.Buffer
	root.SetOut(&buf)
	root.SetErr(&buf)
	root.SetArgs(append([]string{"envsetup"}, args...))
	err := root.Execute()
	// Mirror production's Execute() wrapper (root.go), which renders the JSON
	// error envelope to stdout on failure so --json error output is testable.
	if err != nil {
		_ = output.RenderErrorTo(root.OutOrStdout(), root.ErrOrStderr(), err)
	}
	return buf.String(), err
}

func TestEnvsetupRun_NoToolchain(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv("CJV_TOOLCHAIN", "")
	config.ResetDefaultSettingsFileCache()
	config.ResetCachedUserHomeDir()

	settings := config.DefaultSettings()
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	t.Chdir(t.TempDir())

	_, err := executeEnvsetup(t)
	var noTC *cjverr.NoToolchainConfiguredError
	assert.ErrorAs(t, err, &noTC)
}

func TestEnvsetupRun_OutputContainsExport(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv("CJV_TOOLCHAIN", "")
	config.ResetDefaultSettingsFileCache()
	config.ResetCachedUserHomeDir()

	setupFakeToolchainForCLI(t, home, "lts-1.0.5")
	// Also create bin dir for cjv
	require.NoError(t, os.MkdirAll(filepath.Join(home, "bin"), 0o755))

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	t.Chdir(t.TempDir())

	out, err := executeEnvsetup(t, "--shell=bash")
	require.NoError(t, err)

	assert.Contains(t, out, "export ")
	assert.Contains(t, out, "lts-1.0.5")
	assert.NotContains(t, out, "CJV_TOOLCHAIN")
	assert.NotContains(t, out, "CJV_RECURSION_COUNT")
}

func TestEnvsetupRunJSONOutputsIngredients(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv("CJV_TOOLCHAIN", "")
	config.ResetDefaultSettingsFileCache()
	config.ResetCachedUserHomeDir()

	tcDir := setupFakeToolchainForCLI(t, home, "lts-1.0.5")
	require.NoError(t, os.MkdirAll(filepath.Join(home, "bin"), 0o755))
	require.NoError(t, componentlib.WriteManifest(tcDir, componentlib.Stdx, []string{"dynamic/libfoo"}))

	settings := config.DefaultSettings()
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	t.Chdir(t.TempDir())

	raw, err := executeEnvsetup(t, "--json", "+lts-1.0.5")
	require.NoError(t, err)

	// Empty list fields must serialize as [] not null so typed consumers of
	// the documented schema can rely on array shapes. A +toolchain argument
	// yields nil Targets/Components, which is exactly the regression risk.
	assert.Contains(t, raw, `"targets":[]`)
	assert.Contains(t, raw, `"components":[]`)

	var got envsetupJSONResult
	require.NoError(t, json.Unmarshal([]byte(raw), &got))
	assert.Equal(t, 1, got.SchemaVersion)
	assert.Equal(t, "lts-1.0.5", got.Toolchain.Name)
	assert.Equal(t, tcDir, got.Toolchain.Root)
	assert.Equal(t, "argument", got.Toolchain.Source)
	assert.Equal(t, home, got.CJV.Home)
	assert.Equal(t, filepath.Join(home, "bin"), got.CJV.Bin)
	assert.Equal(t, tcDir, got.Env.Vars["CANGJIE_HOME"])
	assert.NotContains(t, got.Env.Vars, "CJV_TOOLCHAIN")
	assert.NotContains(t, got.Env.Vars, "CJV_RECURSION_COUNT")
	assert.Contains(t, got.Env.Vars, componentlib.EnvStdxDynamic)
	assert.Contains(t, got.Env.Path.Prepend, filepath.Join(tcDir, "bin"))
	userHome, err := os.UserHomeDir()
	require.NoError(t, err)
	assert.Contains(t, got.Env.Path.Append, filepath.Join(userHome, ".cjpm", "bin"))

	if runtime.GOOS == "windows" {
		assert.Nil(t, got.Env.LibraryPath.Key)
		assert.Empty(t, got.Env.LibraryPath.Prepend)
	} else {
		require.NotNil(t, got.Env.LibraryPath.Key)
		assert.True(t, *got.Env.LibraryPath.Key == "LD_LIBRARY_PATH" || *got.Env.LibraryPath.Key == "DYLD_LIBRARY_PATH")
		assert.NotEmpty(t, got.Env.LibraryPath.Prepend)
	}
}

func TestEnvsetupRunJSONUsesDefaultSource(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv("CJV_TOOLCHAIN", "")
	config.ResetDefaultSettingsFileCache()
	config.ResetCachedUserHomeDir()

	setupFakeToolchainForCLI(t, home, "lts-1.0.5")
	require.NoError(t, os.MkdirAll(filepath.Join(home, "bin"), 0o755))

	settings := config.DefaultSettings()
	settings.DefaultToolchain = "lts-1.0.5"
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	// Isolate cwd so toolchain-file resolution can't walk up into an ancestor
	// cangjie-sdk.toml and flip the source away from "default".
	t.Chdir(t.TempDir())

	raw, err := executeEnvsetup(t, "--json")
	require.NoError(t, err)

	var got envsetupJSONResult
	require.NoError(t, json.Unmarshal([]byte(raw), &got))
	assert.Equal(t, "default", got.Toolchain.Source)
	assert.False(t, strings.Contains(raw, "CJV_RECURSION_COUNT"))
}

func TestEnvsetupRunJSONErrorEnvelope(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	t.Setenv("CJV_TOOLCHAIN", "")
	config.ResetDefaultSettingsFileCache()
	config.ResetCachedUserHomeDir()

	settings := config.DefaultSettings()
	require.NoError(t, config.SaveSettings(&settings, filepath.Join(home, ".cjv", "settings.toml")))

	t.Chdir(t.TempDir())

	// A --json failure must emit the machine-readable {"error":{...}} envelope
	// on stdout (rendered by the root command's RenderErrorTo), not just exit
	// non-zero with empty output.
	raw, err := executeEnvsetup(t, "--json")
	require.Error(t, err)

	var envelope struct {
		Error struct {
			Code    string         `json:"code"`
			Message string         `json:"message"`
			Details map[string]any `json:"details"`
		} `json:"error"`
	}
	require.NoError(t, json.Unmarshal([]byte(raw), &envelope))
	assert.Equal(t, "NO_TOOLCHAIN_CONFIGURED", envelope.Error.Code)
	assert.NotEmpty(t, envelope.Error.Message)
}
