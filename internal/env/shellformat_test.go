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
	assert.Equal(t, "$env:PATH = 'C:\\new;C:\\old'\n", result)
}

func TestFormatEnvDiff_Cmd(t *testing.T) {
	diff := []EnvDiff{
		{Key: "PATH", Value: "C:\\new;C:\\old"},
	}
	result := FormatEnvDiff(diff, ShellCmd)
	assert.Equal(t, "set \"PATH=C:\\new;C:\\old\"\n", result)
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
	assert.Equal(t, "$env:FOO = 'value with \"quotes\"'\n", result)
}

func TestFormatEnvDiff_EscapesShellExpansion(t *testing.T) {
	assert.Equal(t,
		"$env:FOO = '$(throw ''pwn'')'\n",
		FormatEnvDiff([]EnvDiff{{Key: "FOO", Value: `$(throw 'pwn')`}}, ShellPowerShell),
	)
	assert.Equal(t,
		"set -gx FOO \"\\$HOME\"\n",
		FormatEnvDiff([]EnvDiff{{Key: "FOO", Value: `$HOME`}}, ShellFish),
	)
	assert.Equal(t,
		"set \"FOO=%%USERNAME%%\"\n",
		FormatEnvDiff([]EnvDiff{{Key: "FOO", Value: `%USERNAME%`}}, ShellCmd),
	)
	assert.Equal(t,
		"set \"FOO=good& echo injected\"\n",
		FormatEnvDiff([]EnvDiff{{Key: "FOO", Value: `good& echo injected`}}, ShellCmd),
	)
}

func TestFormatEnvDiffSkipsUnsafeKeys(t *testing.T) {
	diff := []EnvDiff{
		{Key: "BAD; touch /tmp/pwn", Value: "x"},
		{Key: "SAFE_KEY", Value: "ok"},
	}

	assert.Equal(t, "export SAFE_KEY=\"ok\"\n", FormatEnvDiff(diff, ShellPosix))
	assert.Equal(t, "set -gx SAFE_KEY \"ok\"\n", FormatEnvDiff(diff, ShellFish))
	assert.Equal(t, "$env:SAFE_KEY = 'ok'\n", FormatEnvDiff(diff, ShellPowerShell))
	assert.Equal(t, "set \"SAFE_KEY=ok\"\n", FormatEnvDiff(diff, ShellCmd))
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

func TestParseShellFlagAndDefaultShellType(t *testing.T) {
	tests := map[string]ShellType{
		"bash":       ShellPosix,
		"zsh":        ShellPosix,
		"sh":         ShellPosix,
		"posix":      ShellPosix,
		"fish":       ShellFish,
		"powershell": ShellPowerShell,
		"pwsh":       ShellPowerShell,
		"cmd":        ShellCmd,
	}
	for input, want := range tests {
		got, err := ParseShellFlag(input)
		assert.NoError(t, err)
		assert.Equal(t, want, got)
	}

	_, err := ParseShellFlag("unknown")
	assert.Error(t, err)
	assert.Contains(t, []ShellType{ShellPosix, ShellPowerShell}, DefaultShellType())
}
