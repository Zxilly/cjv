package env

import (
	"os"
	"path/filepath"
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

func TestWritePowerShellEnvScript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "env.ps1")
	binDir := `C:\Users\testuser\.cjv\bin`

	if err := WritePowerShellEnvScript(path, binDir); err != nil {
		t.Fatalf("WritePowerShellEnvScript failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)

	if !strings.Contains(s, binDir) {
		t.Errorf("env.ps1 does not contain binDir %q", binDir)
	}
	if !strings.Contains(s, "$env:PATH") {
		t.Error("env.ps1 missing PATH modification")
	}
}

func TestWriteEnvScripts(t *testing.T) {
	dir := t.TempDir()
	binDir := "/home/testuser/.cjv/bin"

	if err := WriteEnvScripts(dir, binDir); err != nil {
		t.Fatalf("WriteEnvScripts failed: %v", err)
	}

	for _, name := range []string{"env", "env.ps1"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("expected %s to exist: %v", name, err)
		}
	}
}
