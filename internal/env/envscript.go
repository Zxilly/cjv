package env

import (
	"fmt"
	"path/filepath"
	"runtime"

	"github.com/Zxilly/cjv/internal/utils"
)

// WritePosixEnvScript writes a POSIX shell env script that adds binDir to PATH.
func WritePosixEnvScript(path, binDir string) error {
	quotedBinDir := shellQuote(binDir, ShellPosix)
	content := fmt.Sprintf(`#!/bin/sh
# cjv shell setup (managed by cjv, do not edit)
case ":${PATH}:" in
    *:%s:*)
        ;;
    *)
        export PATH=%s:"$PATH"
        ;;
esac
`, quotedBinDir, quotedBinDir)
	return utils.WriteFileAtomic(path, []byte(content), 0o644)
}

// WritePowerShellEnvScript writes a PowerShell env script that prepends the cjv
// bin directory to PATH.
//
// The bin directory is derived at runtime from the script's own location
// (Join-Path $PSScriptRoot 'bin') rather than embedded as a literal. The script
// always lives at <CJV_HOME>/env.ps1 with bin at <CJV_HOME>/bin, so this is
// exact. Keeping the body pure ASCII matters: the file is written UTF-8 without
// a BOM, and Windows PowerShell 5.1 reads BOM-less scripts in the legacy ANSI
// code page — so a non-ASCII home path (e.g. a Chinese Windows username)
// embedded as a literal would be corrupted into a wrong $cjvBin.
func WritePowerShellEnvScript(path string) error {
	const content = `# cjv shell setup (managed by cjv, do not edit)
$cjvBin = Join-Path $PSScriptRoot 'bin'
if (-not ($env:PATH -split [IO.Path]::PathSeparator | Where-Object { $_ -eq $cjvBin })) {
    $env:PATH = "$cjvBin$([IO.Path]::PathSeparator)$env:PATH"
}
`
	return utils.WriteFileAtomic(path, []byte(content), 0o644)
}

// WriteBatEnvScript writes a CMD batch env script that prepends the cjv bin
// directory to PATH.
//
// Like the PowerShell script, the bin directory is derived from the script's
// own location (%~dp0bin) instead of being embedded as a literal, keeping the
// file pure ASCII. CMD reads a .bat in the console's active code page (e.g.
// CP936 on a Chinese system), which would mangle a non-ASCII home path written
// as UTF-8; %~dp0 sidesteps that since CMD already holds the path correctly.
func WriteBatEnvScript(path string) error {
	const content = `@echo off
rem cjv shell setup (managed by cjv, do not edit)
set "cjvBin=%~dp0bin"
set "cjvFound="
for %%P in ("%PATH:;=" "%") do (
    if /I "%%~P"=="%cjvBin%" set "cjvFound=1"
)
if not defined cjvFound (
    set "PATH=%cjvBin%;%PATH%"
)
`
	return utils.WriteFileAtomic(path, []byte(content), 0o644)
}

// WriteEnvScripts writes platform-appropriate env scripts to the given directory.
// On Windows it generates env.ps1 and env.bat; on other platforms it generates env (POSIX).
// The caller must ensure homeDir exists (e.g. via config.EnsureDirs).
func WriteEnvScripts(homeDir, binDir string) error {
	if runtime.GOOS == "windows" {
		if err := WritePowerShellEnvScript(filepath.Join(homeDir, "env.ps1")); err != nil {
			return err
		}
		return WriteBatEnvScript(filepath.Join(homeDir, "env.bat"))
	}
	return WritePosixEnvScript(filepath.Join(homeDir, "env"), binDir)
}
