package env

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDiffEnvSkipsVolatileVars(t *testing.T) {
	before := map[string]string{
		"PATH": "/usr/bin",
	}
	after := map[string]string{
		"PATH":         "/sdk/bin:/usr/bin",
		"PWD":          "/sdk",
		"SHLVL":        "2",
		"_":            "/usr/bin/env",
		"CANGJIE_HOME": "/sdk",
	}

	cfg := diffEnv(before, after)
	assert.Equal(t, "/sdk", cfg.Vars["CANGJIE_HOME"])
	assert.NotContains(t, cfg.Vars, "PWD")
	assert.NotContains(t, cfg.Vars, "SHLVL")
	assert.NotContains(t, cfg.Vars, "_")
}

func TestCaptureEnvSetupReturnsErrorWhenScriptFails(t *testing.T) {
	sdkDir := t.TempDir()
	var scriptPath string
	var contents string
	if runtime.GOOS == "windows" {
		scriptPath = filepath.Join(sdkDir, "envsetup.ps1")
		contents = "throw 'boom'"
	} else {
		scriptPath = filepath.Join(sdkDir, "envsetup.sh")
		contents = "#!/bin/sh\nexit 1\n"
	}
	require.NoError(t, os.WriteFile(scriptPath, []byte(contents), 0o755))

	_, err := CaptureEnvSetup(context.Background(), sdkDir)
	require.Error(t, err)
}

func TestCaptureEnvSetup_MissingScriptReturnsEmpty(t *testing.T) {
	// An SDK directory without an envsetup script is valid (e.g., minimal SDK).
	// CaptureEnvSetup should return an empty config, not an error.
	sdkDir := t.TempDir()

	cfg, err := CaptureEnvSetup(context.Background(), sdkDir)
	require.NoError(t, err, "missing script should not be an error")
	assert.Empty(t, cfg.Vars, "should have no captured vars")
	assert.Empty(t, cfg.PathPrepend.Entries, "should have no PATH additions")
}

func TestDiffEnv_CapturesNewVariables(t *testing.T) {
	// envsetup.sh sets CANGJIE_HOME and CANGJIE_VERSION.
	// These should appear in the diff.
	before := map[string]string{
		"HOME": "/home/user",
	}
	after := map[string]string{
		"HOME":            "/home/user",
		"CANGJIE_HOME":    "/opt/sdk/1.0.5",
		"CANGJIE_VERSION": "1.0.5",
	}

	result := diffEnv(before, after)
	assert.Equal(t, "/opt/sdk/1.0.5", result.Vars["CANGJIE_HOME"])
	assert.Equal(t, "1.0.5", result.Vars["CANGJIE_VERSION"])
}

func TestDiffEnv_IgnoresUnchangedVariables(t *testing.T) {
	// Variables that are the same before and after the script should
	// not be captured -- they're not the SDK's doing.
	before := map[string]string{
		"HOME":   "/home/user",
		"EDITOR": "vim",
	}
	after := map[string]string{
		"HOME":         "/home/user",
		"EDITOR":       "vim",
		"CANGJIE_HOME": "/opt/sdk",
	}

	result := diffEnv(before, after)
	assert.NotContains(t, result.Vars, "HOME")
	assert.NotContains(t, result.Vars, "EDITOR")
	assert.Contains(t, result.Vars, "CANGJIE_HOME")
}

func TestDiffEnv_ExtractsOnlyNewPathEntries(t *testing.T) {
	// The SDK's envsetup prepends its bin directory to PATH.
	// diffEnv should capture only the new entries, not the full PATH.
	sep := string(os.PathListSeparator)
	before := map[string]string{
		"PATH": strings.Join([]string{"/usr/bin", "/usr/local/bin"}, sep),
	}
	after := map[string]string{
		"PATH": strings.Join([]string{"/opt/sdk/bin", "/usr/bin", "/usr/local/bin"}, sep),
	}

	result := diffEnv(before, after)
	assert.Contains(t, result.PathPrepend.Entries, "/opt/sdk/bin",
		"newly added PATH entry should be captured")
	// PATH should not appear as a regular Var since it gets special treatment
	assert.NotContains(t, result.Vars, "PATH")
}

func TestDiffEnv_FiltersVolatileVars(t *testing.T) {
	// Variables like PWD, SHLVL change as a side effect of running a shell.
	// They must not be captured as SDK configuration.
	before := map[string]string{}
	after := map[string]string{
		"PWD":          "/tmp",
		"SHLVL":        "2",
		"_":            "/bin/bash",
		"_CJV_SCRIPT":  "/opt/sdk/envsetup.sh",
		"CANGJIE_HOME": "/opt/sdk",
	}

	result := diffEnv(before, after)
	assert.NotContains(t, result.Vars, "PWD")
	assert.NotContains(t, result.Vars, "SHLVL")
	assert.NotContains(t, result.Vars, "_")
	assert.NotContains(t, result.Vars, "_CJV_SCRIPT")
	assert.Contains(t, result.Vars, "CANGJIE_HOME")
}
