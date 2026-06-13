package env

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWritePosixEnvScript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "env")
	binDir := "/home/testuser/.cjv/bin"

	if err := WritePosixEnvScript(path, binDir); err != nil {
		t.Fatalf("WritePosixEnvScript failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)

	if !strings.Contains(s, binDir) {
		t.Errorf("env script does not contain binDir %q", binDir)
	}
	if !strings.Contains(s, "export PATH=") {
		t.Error("env script missing PATH export")
	}
	if !strings.Contains(s, `case ":${PATH}:"`) {
		t.Error("env script missing duplicate check")
	}
}

func TestWritePosixEnvScriptEscapesBinDir(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "env")
	binDir := `/tmp/cjv"; touch /tmp/pwn; echo "$HOME/bin`

	if err := WritePosixEnvScript(path, binDir); err != nil {
		t.Fatalf("WritePosixEnvScript failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)

	if strings.Contains(s, `"/tmp/cjv"; touch /tmp/pwn`) {
		t.Fatal("env script contains an unescaped double quote that can terminate the PATH literal")
	}
	if !strings.Contains(s, `\"; touch /tmp/pwn`) {
		t.Fatal("env script did not escape embedded double quotes in binDir")
	}
	if !strings.Contains(s, `\$HOME`) {
		t.Fatal("env script did not escape dollar expansion in binDir")
	}
}

func TestWritePowerShellEnvScript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "env.ps1")

	if err := WritePowerShellEnvScript(path); err != nil {
		t.Fatalf("WritePowerShellEnvScript failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)

	if !strings.Contains(s, `$cjvBin = Join-Path $PSScriptRoot 'bin'`) {
		t.Error("env.ps1 does not derive the bin dir from $PSScriptRoot")
	}
	if !strings.Contains(s, "$env:PATH") {
		t.Error("env.ps1 missing PATH modification")
	}
}

func TestWriteBatEnvScript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "env.bat")

	if err := WriteBatEnvScript(path); err != nil {
		t.Fatalf("WriteBatEnvScript failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)

	if !strings.Contains(s, `set "cjvBin=%~dp0bin"`) {
		t.Error("env.bat does not derive the bin dir from %~dp0")
	}
	if !strings.Contains(s, `%PATH%`) {
		t.Error("env.bat missing PATH reference")
	}
}

func TestWriteBatEnvScriptChecksExactPathEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "env.bat")

	if err := WriteBatEnvScript(path); err != nil {
		t.Fatalf("WriteBatEnvScript failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)

	if strings.Contains(s, "find /I") {
		t.Fatal("env.bat uses substring matching instead of exact PATH entry matching")
	}
	if !strings.Contains(s, `for %%P in ("%PATH:;=" "%") do (`) {
		t.Fatal("env.bat does not iterate over PATH entries before checking for duplicates")
	}
	if !strings.Contains(s, `if /I "%%~P"=="%cjvBin%"`) {
		t.Fatal("env.bat does not compare complete PATH entries")
	}
}

// TestWindowsEnvScriptsAreSelfLocatingASCII guards the fix for non-ASCII home
// paths (e.g. a Chinese Windows username). The Windows env scripts must derive
// the cjv bin directory from their own on-disk location instead of embedding it
// as a literal, so their bytes stay pure ASCII. A UTF-8 file carrying a
// non-ASCII path literal gets corrupted when CMD reads the .bat in CP936 or
// Windows PowerShell 5.1 reads the BOM-less .ps1 as legacy ANSI, leaving a
// wrong bin path that never matches and silently breaks current-session PATH.
func TestWindowsEnvScriptsAreSelfLocatingASCII(t *testing.T) {
	dir := t.TempDir()

	ps1 := filepath.Join(dir, "env.ps1")
	if err := WritePowerShellEnvScript(ps1); err != nil {
		t.Fatalf("WritePowerShellEnvScript failed: %v", err)
	}
	bat := filepath.Join(dir, "env.bat")
	if err := WriteBatEnvScript(bat); err != nil {
		t.Fatalf("WriteBatEnvScript failed: %v", err)
	}

	cases := []struct {
		path   string
		marker string // the runtime self-location expression the script must use
	}{
		{ps1, `Join-Path $PSScriptRoot 'bin'`},
		{bat, `set "cjvBin=%~dp0bin"`},
	}
	for _, c := range cases {
		name := filepath.Base(c.path)
		content, err := os.ReadFile(c.path)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(content), c.marker) {
			t.Errorf("%s does not self-locate the bin dir via %q", name, c.marker)
		}
		for i, b := range content {
			if b > 0x7F {
				t.Errorf("%s has a non-ASCII byte at offset %d; a non-ASCII home path would corrupt the script", name, i)
				break
			}
		}
	}
}

func TestWriteEnvScripts(t *testing.T) {
	dir := t.TempDir()
	binDir := "/home/testuser/.cjv/bin"

	if err := WriteEnvScripts(dir, binDir); err != nil {
		t.Fatalf("WriteEnvScripts failed: %v", err)
	}

	var expected, unexpected []string
	if runtime.GOOS == "windows" {
		expected = []string{"env.ps1", "env.bat"}
		unexpected = []string{"env"}
	} else {
		expected = []string{"env"}
		unexpected = []string{"env.ps1", "env.bat"}
	}

	for _, name := range expected {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("expected %s to exist: %v", name, err)
		}
	}
	for _, name := range unexpected {
		if _, err := os.Stat(filepath.Join(dir, name)); !os.IsNotExist(err) {
			t.Errorf("unexpected %s exists on this platform", name)
		}
	}
}
