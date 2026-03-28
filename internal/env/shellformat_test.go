package env

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatEnvDiff_Posix(t *testing.T) {
	diff := []EnvDiff{
		{Key: "PATH", Value: "/new/path:/old/path"},
		{Key: "LD_LIBRARY_PATH", Value: "/sdk/lib"},
	}
	result := FormatEnvDiff(diff, ShellPosix)
	assert.Equal(t, "export PATH=\"/new/path:/old/path\"\nexport LD_LIBRARY_PATH=\"/sdk/lib\"\n", result)
}

func TestFormatEnvDiff_Fish(t *testing.T) {
	diff := []EnvDiff{
		{Key: "PATH", Value: "/new/path:/old/path"},
	}
	result := FormatEnvDiff(diff, ShellFish)
	assert.Equal(t, "set -gx PATH \"/new/path:/old/path\"\n", result)
}

func TestFormatEnvDiff_PowerShell(t *testing.T) {
	diff := []EnvDiff{
		{Key: "PATH", Value: "C:\\new;C:\\old"},
	}
	result := FormatEnvDiff(diff, ShellPowerShell)
	assert.Equal(t, "$env:PATH = \"C:\\new;C:\\old\"\n", result)
}

func TestFormatEnvDiff_Cmd(t *testing.T) {
	diff := []EnvDiff{
		{Key: "PATH", Value: "C:\\new;C:\\old"},
	}
	result := FormatEnvDiff(diff, ShellCmd)
	assert.Equal(t, "set PATH=C:\\new;C:\\old\n", result)
}

func TestFormatEnvDiff_PosixEscaping(t *testing.T) {
	diff := []EnvDiff{
		{Key: "FOO", Value: `value with "quotes" and $dollar`},
	}
	result := FormatEnvDiff(diff, ShellPosix)
	assert.Equal(t, "export FOO=\"value with \\\"quotes\\\" and \\$dollar\"\n", result)
}

func TestFormatEnvDiff_PowerShellEscaping(t *testing.T) {
	diff := []EnvDiff{
		{Key: "FOO", Value: `value with "quotes"`},
	}
	result := FormatEnvDiff(diff, ShellPowerShell)
	assert.Equal(t, "$env:FOO = \"value with `\"quotes`\"\"\n", result)
}

func TestComputeEnvDiff(t *testing.T) {
	base := []string{"PATH=/usr/bin", "HOME=/home/user", "UNCHANGED=same"}
	modified := []string{"PATH=/new:/usr/bin", "HOME=/home/user", "UNCHANGED=same", "NEW_VAR=hello"}
	diff := ComputeEnvDiff(base, modified)

	assert.Len(t, diff, 2)
	keys := make(map[string]string)
	for _, d := range diff {
		keys[d.Key] = d.Value
	}
	assert.Equal(t, "/new:/usr/bin", keys["PATH"])
	assert.Equal(t, "hello", keys["NEW_VAR"])
}
