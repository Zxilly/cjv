package env

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// isCaptureUnavailableError returns true for errors that indicate the
// capture mechanism is not available (missing script or missing shell).
func isCaptureUnavailableError(err error) bool {
	return errors.Is(err, os.ErrNotExist) || errors.Is(err, exec.ErrNotFound)
}

// CaptureEnvSetup executes the SDK's envsetup script and captures
// environment variable differences.
func CaptureEnvSetup(ctx context.Context, sdkDir string) (*EnvConfig, error) {
	cfg, err := CaptureEnvSetupStrict(ctx, sdkDir)
	if err != nil {
		if isCaptureUnavailableError(err) {
			return NewEnvConfig(), nil
		}
		return nil, err
	}
	return cfg, nil
}

// CaptureEnvSetupStrict executes the SDK's envsetup script and returns an
// error when capture prerequisites are unavailable.
func CaptureEnvSetupStrict(ctx context.Context, sdkDir string) (*EnvConfig, error) {
	baseEnv := os.Environ()
	before := envMapFromSlice(baseEnv)

	var after map[string]string
	var err error

	switch runtime.GOOS {
	case "windows":
		after, err = captureWindows(ctx, sdkDir, baseEnv)
	default:
		after, err = captureUnix(ctx, sdkDir, baseEnv)
	}

	if err != nil {
		return nil, err
	}

	return diffEnv(before, after), nil
}

func captureUnix(ctx context.Context, sdkDir string, baseEnv []string) (map[string]string, error) {
	script := filepath.Join(sdkDir, "envsetup.sh")
	if _, err := os.Stat(script); err != nil {
		return nil, err
	}
	// Pass script path via env var to avoid shell injection from special chars in paths
	// Use --norc --noprofile to prevent user profile scripts from contaminating
	// the environment diff with variables unrelated to the SDK.
	cmd := exec.CommandContext(ctx, "bash", "--norc", "--noprofile", "-c", `source "$_CJV_SCRIPT" && env -0`)
	cmd.Dir = sdkDir
	cmd.Env = append(baseEnv, "_CJV_SCRIPT="+script)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	entries := strings.Split(string(out), "\x00")
	return envMapFromSlice(entries), nil
}

func captureWindows(ctx context.Context, sdkDir string, baseEnv []string) (map[string]string, error) {
	script := filepath.Join(sdkDir, "envsetup.ps1")
	if _, err := os.Stat(script); err != nil {
		return nil, err
	}
	// Pass script path via env var to avoid injection from special chars in paths
	// Use -NoProfile to prevent user profile scripts from contaminating the diff.
	// Force UTF-8 output to avoid UTF-16LE encoding issues with non-ASCII env values.
	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-Command",
		`[Console]::OutputEncoding = [Text.Encoding]::UTF8; . $env:_CJV_SCRIPT; [System.Environment]::GetEnvironmentVariables().GetEnumerator() | ForEach-Object { "$($_.Key)=$($_.Value)" + [char]0 }`)
	cmd.Dir = sdkDir
	cmd.Env = append(baseEnv, "_CJV_SCRIPT="+script)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	entries := strings.Split(string(out), "\x00")
	return envMapFromSlice(entries), nil
}

func envMapFromSlice(envSlice []string) map[string]string {
	m := make(map[string]string, len(envSlice))
	for _, e := range envSlice {
		e = strings.Trim(e, "\r\n")
		k, v, ok := strings.Cut(e, "=")
		if ok && k != "" {
			m[k] = v
		}
	}
	return m
}

func diffEnv(before, after map[string]string) *EnvConfig {
	cfg := NewEnvConfig()
	pathSep := string(os.PathListSeparator)

	// On Windows, env var names are case-insensitive — build a lookup map
	beforeCI := before
	if runtime.GOOS == "windows" {
		beforeCI = make(map[string]string, len(before))
		for bk, bv := range before {
			beforeCI[canonicalEnvKey(bk)] = bv
		}
	}

	for k, v := range after {
		if isVolatileEnvVar(k) {
			continue
		}
		if strings.EqualFold(k, "PATH") {
			// PATH gets special treatment: extract newly added entries
			lookupKey := canonicalEnvKey(k)
			oldEntries := strings.Split(beforeCI[lookupKey], pathSep)
			newEntries := strings.Split(v, pathSep)
			oldSet := make(map[string]bool, len(oldEntries))
			for _, e := range oldEntries {
				if runtime.GOOS == "windows" {
					oldSet[strings.ToLower(e)] = true
				} else {
					oldSet[e] = true
				}
			}
			for _, e := range newEntries {
				entryKey := e
				if runtime.GOOS == "windows" {
					entryKey = strings.ToLower(e)
				}
				if !oldSet[entryKey] && e != "" {
					cfg.PathPrepend.Entries = append(cfg.PathPrepend.Entries, e)
				}
			}
			continue
		}

		// Compare with before value (case-insensitive key on Windows)
		lookupKey := canonicalEnvKey(k)
		if beforeCI[lookupKey] != v {
			cfg.Vars[k] = v
		}
	}
	return cfg
}

func isVolatileEnvVar(key string) bool {
	switch strings.ToUpper(key) {
	case "PWD", "OLDPWD", "SHLVL", "_", "_CJV_SCRIPT":
		return true
	default:
		return false
	}
}
