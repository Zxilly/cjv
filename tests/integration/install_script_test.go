//go:build integration

package integration

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// scriptPath returns the absolute path to a file in the web/ directory.
func scriptPath(t *testing.T, name string) string {
	t.Helper()
	p, err := filepath.Abs(filepath.Join("..", "..", "web", "public", name))
	require.NoError(t, err)
	return p
}

// ---------- Unix: install.sh ----------

func TestIntegrationInstallScriptSh(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script tests are for Unix")
	}

	server := mockCJVDownloadServer(t)
	cjvHome := t.TempDir()
	fakeHome := t.TempDir()

	cmd := exec.Command("sh", scriptPath(t, "install.sh"),
		"-y", "--default-toolchain", "none", "--no-modify-path")
	cmd.Env = installScriptEnv(server.URL, cjvHome, "HOME="+fakeHome)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "install.sh failed:\n%s", string(out))

	assert.FileExists(t, filepath.Join(cjvHome, "bin", "cjv"))
	for _, tool := range []string{"cjc", "cjpm"} {
		assert.FileExists(t, filepath.Join(cjvHome, "bin", tool))
	}
	assert.FileExists(t, filepath.Join(cjvHome, "env"))
}

func TestIntegrationInstallScriptShWithToolchain(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script tests are for Unix")
	}

	server := mockCJVDownloadServerWithSDK(t)
	cjvHome := t.TempDir()
	fakeHome := t.TempDir()
	writeIntegrationSettings(t, cjvHome, server.URL)

	cmd := exec.Command("sh", scriptPath(t, "install.sh"),
		"-y", "--no-modify-path")
	cmd.Env = installScriptEnv(server.URL, cjvHome, "HOME="+fakeHome)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "install.sh with toolchain failed:\n%s", string(out))

	assert.DirExists(t, filepath.Join(cjvHome, "toolchains", "lts-1.0.5"))
	for _, tool := range proxy.AllProxyTools() {
		assert.FileExists(t, filepath.Join(cjvHome, "bin", tool))
	}
}

// ---------- Windows: install.ps1 ----------

func TestIntegrationInstallScriptPs1(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("PS1 script tests are for Windows")
	}

	server := mockCJVDownloadServer(t)
	cjvHome := t.TempDir()

	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass",
		"-File", scriptPath(t, "install.ps1"),
		"-Yes", "-DefaultToolchain", "none", "-NoModifyPath")
	cmd.Env = installScriptEnv(server.URL, cjvHome)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "install.ps1 failed:\n%s", string(out))

	assert.FileExists(t, filepath.Join(cjvHome, "bin", "cjv.exe"))
	for _, tool := range []string{"cjc", "cjpm"} {
		assert.FileExists(t, filepath.Join(cjvHome, "bin", tool+".exe"))
	}
	assert.FileExists(t, filepath.Join(cjvHome, "env.ps1"))
	assert.FileExists(t, filepath.Join(cjvHome, "env.bat"))
}

func TestIntegrationInstallScriptPs1WithToolchain(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("PS1 script tests are for Windows")
	}

	server := mockCJVDownloadServerWithSDK(t)
	cjvHome := t.TempDir()
	writeIntegrationSettings(t, cjvHome, server.URL)

	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass",
		"-File", scriptPath(t, "install.ps1"),
		"-Yes", "-NoModifyPath")
	cmd.Env = installScriptEnv(server.URL, cjvHome)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "install.ps1 with toolchain failed:\n%s", string(out))

	assert.DirExists(t, filepath.Join(cjvHome, "toolchains", "lts-1.0.5"))
	for _, tool := range proxy.AllProxyTools() {
		assert.FileExists(t, filepath.Join(cjvHome, "bin", tool+".exe"))
	}
}

// ---------- Proxy execution tests ----------

// runProxyTool executes a proxy tool binary from CJV_HOME/bin/ and returns its output.
func runProxyTool(t *testing.T, cjvHome, toolName string) (string, error) {
	t.Helper()
	binName := toolName
	if runtime.GOOS == "windows" {
		binName += ".exe"
	}

	cmd := exec.Command(filepath.Join(cjvHome, "bin", binName))
	cmd.Env = installScriptEnv("", cjvHome)
	out, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(out)), err
}

func TestIntegrationInstallScriptShProxyExecution(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script tests are for Unix")
	}

	server := mockCJVDownloadServerWithExecutableSDK(t)
	cjvHome := t.TempDir()
	fakeHome := t.TempDir()
	writeIntegrationSettings(t, cjvHome, server.URL)

	cmd := exec.Command("sh", scriptPath(t, "install.sh"),
		"-y", "--no-modify-path")
	cmd.Env = installScriptEnv(server.URL, cjvHome, "HOME="+fakeHome)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "install.sh failed:\n%s", string(out))

	output, err := runProxyTool(t, cjvHome, "cjc")
	require.NoError(t, err, "cjc proxy execution failed: %s", output)
	assert.Contains(t, output, "cjc stub")

	output, err = runProxyTool(t, cjvHome, "cjpm")
	require.NoError(t, err, "cjpm proxy execution failed: %s", output)
	assert.Contains(t, output, "cjpm stub")
}

func TestIntegrationInstallScriptPs1ProxyExecution(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("PS1 script tests are for Windows")
	}

	server := mockCJVDownloadServerWithExecutableSDK(t)
	cjvHome := t.TempDir()
	writeIntegrationSettings(t, cjvHome, server.URL)

	cmd := exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass",
		"-File", scriptPath(t, "install.ps1"),
		"-Yes", "-NoModifyPath")
	cmd.Env = installScriptEnv(server.URL, cjvHome)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "install.ps1 failed:\n%s", string(out))

	output, err := runProxyTool(t, cjvHome, "cjc")
	require.NoError(t, err, "cjc proxy execution failed: %s", output)
	assert.Contains(t, output, "cjc stub")

	output, err = runProxyTool(t, cjvHome, "cjpm")
	require.NoError(t, err, "cjpm proxy execution failed: %s", output)
	assert.Contains(t, output, "cjpm stub")
}
