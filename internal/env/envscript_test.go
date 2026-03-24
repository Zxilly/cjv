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

func TestWriteBatEnvScript(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "env.bat")
	binDir := `C:\Users\testuser\.cjv\bin`

	if err := WriteBatEnvScript(path, binDir); err != nil {
		t.Fatalf("WriteBatEnvScript failed: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(content)

	if !strings.Contains(s, binDir) {
		t.Errorf("env.bat does not contain binDir %q", binDir)
	}
	if !strings.Contains(s, `%PATH%`) {
		t.Error("env.bat missing PATH reference")
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
