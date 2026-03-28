package env

import (
	"fmt"
	"runtime"
	"strings"
)

// ShellType represents a shell environment type.
type ShellType int

const (
	ShellPosix      ShellType = iota // bash, zsh, sh
	ShellFish                        // fish
	ShellPowerShell                  // powershell, pwsh
	ShellCmd                         // cmd.exe
)

// EnvDiff represents a single environment variable change.
type EnvDiff struct {
	Key   string
	Value string
}

// ComputeEnvDiff compares base and modified environments, returning only changed/new entries.
func ComputeEnvDiff(base, modified []string) []EnvDiff {
	baseMap := make(map[string]string, len(base))
	for _, e := range base {
		k, v, ok := strings.Cut(e, "=")
		if !ok || k == "" {
			continue
		}
		baseMap[canonicalEnvKey(k)] = v
	}

	var diffs []EnvDiff
	for _, e := range modified {
		k, v, ok := strings.Cut(e, "=")
		if !ok || k == "" {
			continue
		}
		canonical := canonicalEnvKey(k)
		if oldVal, exists := baseMap[canonical]; exists && oldVal == v {
			continue
		}
		diffs = append(diffs, EnvDiff{Key: k, Value: v})
	}
	return diffs
}

// FormatEnvDiff formats environment diffs as shell-specific commands.
func FormatEnvDiff(diffs []EnvDiff, shell ShellType) string {
	var b strings.Builder
	for _, d := range diffs {
		switch shell {
		case ShellFish:
			fmt.Fprintf(&b, "set -gx %s %s\n", d.Key, shellQuote(d.Value, shell))
		case ShellPowerShell:
			fmt.Fprintf(&b, "$env:%s = %s\n", d.Key, shellQuote(d.Value, shell))
		case ShellCmd:
			fmt.Fprintf(&b, "set %s=%s\n", d.Key, d.Value)
		default: // ShellPosix
			fmt.Fprintf(&b, "export %s=%s\n", d.Key, shellQuote(d.Value, shell))
		}
	}
	return b.String()
}

func shellQuote(value string, shell ShellType) string {
	switch shell {
	case ShellCmd:
		return value
	case ShellPowerShell:
		// PowerShell: escape " with `"
		escaped := strings.ReplaceAll(value, `"`, "`\"")
		return `"` + escaped + `"`
	case ShellFish:
		// Fish: escape \ and "
		escaped := strings.ReplaceAll(value, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		return `"` + escaped + `"`
	default: // ShellPosix
		// POSIX: escape \, ", $, `
		escaped := strings.ReplaceAll(value, `\`, `\\`)
		escaped = strings.ReplaceAll(escaped, `"`, `\"`)
		escaped = strings.ReplaceAll(escaped, `$`, `\$`)
		escaped = strings.ReplaceAll(escaped, "`", "\\`")
		return `"` + escaped + `"`
	}
}

// ParseShellFlag parses the --shell flag value into a ShellType.
func ParseShellFlag(s string) (ShellType, error) {
	switch strings.ToLower(s) {
	case "bash", "zsh", "sh", "posix":
		return ShellPosix, nil
	case "fish":
		return ShellFish, nil
	case "powershell", "pwsh":
		return ShellPowerShell, nil
	case "cmd":
		return ShellCmd, nil
	default:
		return ShellPosix, fmt.Errorf("unknown shell type: %s (supported: bash, fish, powershell, cmd)", s)
	}
}

// DefaultShellType returns the default shell type for the current platform.
func DefaultShellType() ShellType {
	if runtime.GOOS == "windows" {
		return ShellPowerShell
	}
	return ShellPosix
}
