package env

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/Zxilly/cjv/internal/utils"
)

// WritePosixEnvScript writes a POSIX shell env script that adds binDir to PATH.
func WritePosixEnvScript(path, binDir string) error {
	content := fmt.Sprintf(`#!/bin/sh
# cjv shell setup (managed by cjv, do not edit)
case ":${PATH}:" in
    *:"%s":*)
        ;;
    *)
        export PATH="%s:$PATH"
        ;;
esac
`, binDir, binDir)
	return utils.WriteFileAtomic(path, []byte(content), 0o644)
}

// WritePowerShellEnvScript writes a PowerShell env script that adds binDir to PATH.
func WritePowerShellEnvScript(path, binDir string) error {
	content := fmt.Sprintf(`# cjv shell setup (managed by cjv, do not edit)
$cjvBin = "%s"
if (-not ($env:PATH -split [IO.Path]::PathSeparator | Where-Object { $_ -eq $cjvBin })) {
    $env:PATH = "$cjvBin$([IO.Path]::PathSeparator)$env:PATH"
}
`, binDir)
	return utils.WriteFileAtomic(path, []byte(content), 0o644)
}

// WriteBatEnvScript writes a CMD batch env script that adds binDir to PATH.
func WriteBatEnvScript(path, binDir string) error {
	content := fmt.Sprintf(`@echo off
rem cjv shell setup (managed by cjv, do not edit)
echo %%PATH%% | find /I "%s" >nul
if errorlevel 1 (
    set "PATH=%s;%%PATH%%"
)
`, binDir, binDir)
	return utils.WriteFileAtomic(path, []byte(content), 0o644)
}

// WriteEnvScripts writes platform-appropriate env scripts to the given directory.
// On Windows it generates env.ps1 and env.bat; on other platforms it generates env (POSIX).
// The caller must ensure homeDir exists (e.g. via config.EnsureDirs).
func WriteEnvScripts(homeDir, binDir string) error {
	if runtime.GOOS == "windows" {
		if err := WritePowerShellEnvScript(filepath.Join(homeDir, "env.ps1"), binDir); err != nil {
			return err
		}
		return WriteBatEnvScript(filepath.Join(homeDir, "env.bat"), binDir)
	}
	return WritePosixEnvScript(filepath.Join(homeDir, "env"), binDir)
}
