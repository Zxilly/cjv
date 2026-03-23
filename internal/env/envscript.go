package env

import (
	"fmt"
	"path/filepath"

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

// WriteEnvScripts writes both env and env.ps1 scripts to the given directory.
// The caller must ensure homeDir exists (e.g. via config.EnsureDirs).
func WriteEnvScripts(homeDir, binDir string) error {
	if err := WritePosixEnvScript(filepath.Join(homeDir, "env"), binDir); err != nil {
		return err
	}
	return WritePowerShellEnvScript(filepath.Join(homeDir, "env.ps1"), binDir)
}
