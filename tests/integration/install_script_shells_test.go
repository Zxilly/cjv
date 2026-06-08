//go:build integration && installscript

package integration

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func installScriptTestEnv(name, fallback string) string {
	if value := os.Getenv(name); value != "" {
		return value
	}
	return fallback
}

func installScriptEnvWithPathSetup(serverURL, cjvHome string) []string {
	env := installScriptEnv(serverURL, cjvHome)
	out := env[:0]
	for _, entry := range env {
		if strings.HasPrefix(entry, "CJV_NO_PATH_SETUP=") {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func runInstallShFromShell(t *testing.T, shellName, script, cjvHome, serverURL string, args ...string) string {
	t.Helper()

	shellPath, err := exec.LookPath(shellName)
	if err != nil {
		t.Skipf("%s not installed: %v", shellName, err)
	}

	var cmd *exec.Cmd
	if shellName == "fish" || filepath.Base(shellPath) == "fish" {
		cmd = exec.Command(shellPath, append([]string{"-c", "sh $argv", script}, args...)...)
	} else {
		cmd = exec.Command(shellPath, append([]string{"-c", `sh "$@"`, "cjv-install", script}, args...)...)
	}
	cmd.Env = installScriptEnvWithPathSetup(serverURL, cjvHome)

	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "install.sh via %s failed:\n%s", shellName, string(out))
	return string(out)
}

func assertUnixShellPathConfig(t *testing.T, shellName, home string) {
	t.Helper()

	binDir := filepath.Join(home, "bin")
	marker := "# cjv (managed by cjv, do not edit)"

	switch shellName {
	case "fish":
		content, err := os.ReadFile(filepath.Join(home, ".config", "fish", "config.fish"))
		require.NoError(t, err)
		assert.Contains(t, string(content), marker)
		assert.Contains(t, string(content), "fish_add_path")
		assert.Contains(t, string(content), binDir)
	case "zsh":
		for _, rc := range []string{".zshrc", ".zprofile"} {
			content, err := os.ReadFile(filepath.Join(home, rc))
			require.NoError(t, err)
			assert.Contains(t, string(content), marker)
			assert.Contains(t, string(content), "export PATH=")
			assert.Contains(t, string(content), binDir)
		}
	default:
		for _, rc := range []string{".profile", ".bashrc"} {
			content, err := os.ReadFile(filepath.Join(home, rc))
			require.NoError(t, err)
			assert.Contains(t, string(content), marker)
			assert.Contains(t, string(content), "export PATH=")
			assert.Contains(t, string(content), binDir)
		}
	}
}

func TestIntegrationInstallScriptShFromConfiguredShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell script tests are for Unix")
	}
	requireCI(t)

	shellName := installScriptTestEnv("CJV_INSTALL_TEST_SHELL", "sh")
	server := mockCJVDownloadServer(t)
	cjvHome := t.TempDir()

	require.NoError(t, os.MkdirAll(filepath.Join(cjvHome, ".config", "fish"), 0o755))
	runInstallShFromShell(t, shellName, scriptPath(t, "install.sh"), cjvHome, server.URL,
		"-y", "--default-toolchain", "none")

	assert.FileExists(t, filepath.Join(cjvHome, "bin", "cjv"))
	for _, tool := range []string{"cjc", "cjpm"} {
		assert.FileExists(t, filepath.Join(cjvHome, "bin", tool))
	}
	assert.FileExists(t, filepath.Join(cjvHome, "env"))
	assertUnixShellPathConfig(t, shellName, cjvHome)
}

func TestIntegrationInstallScriptPs1FromConfiguredPowerShell(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("PS1 script tests are for Windows")
	}

	shellName := installScriptTestEnv("CJV_INSTALL_TEST_POWERSHELL", "powershell")
	defaultToolchain := installScriptTestEnv("CJV_INSTALL_TEST_DEFAULT_TOOLCHAIN", "all")

	shellPath, err := exec.LookPath(shellName)
	if err != nil {
		t.Skipf("%s not installed: %v", shellName, err)
	}

	modes := []string{defaultToolchain}
	if defaultToolchain == "all" {
		modes = []string{"none", "lts"}
	}
	for _, mode := range modes {
		t.Run(mode, func(t *testing.T) {
			runInstallPs1WithToolchainMode(t, shellName, shellPath, mode)
		})
	}
}

func runInstallPs1WithToolchainMode(t *testing.T, shellName, shellPath, defaultToolchain string) {
	t.Helper()

	cjvHome := t.TempDir()
	var serverURL string
	args := []string{
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-File", scriptPath(t, "install.ps1"),
		"-Yes",
		"-NoModifyPath",
	}

	switch defaultToolchain {
	case "none":
		server := mockCJVDownloadServer(t)
		serverURL = server.URL
		args = append(args, "-DefaultToolchain", "none")
	case "lts":
		server := mockCJVDownloadServerWithSDK(t)
		serverURL = server.URL
		writeIntegrationSettings(t, cjvHome, server.URL)
	default:
		t.Fatalf("unsupported CJV_INSTALL_TEST_DEFAULT_TOOLCHAIN %q", defaultToolchain)
	}

	cmd := exec.Command(shellPath, args...)
	cmd.Env = installScriptEnv(serverURL, cjvHome)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "install.ps1 via %s failed:\n%s", shellName, string(out))

	assert.FileExists(t, filepath.Join(cjvHome, "bin", "cjv.exe"))
	for _, tool := range []string{"cjc", "cjpm"} {
		assert.FileExists(t, filepath.Join(cjvHome, "bin", tool+".exe"))
	}
	assert.FileExists(t, filepath.Join(cjvHome, "env.ps1"))
	assert.FileExists(t, filepath.Join(cjvHome, "env.bat"))

	if defaultToolchain == "lts" {
		assert.DirExists(t, filepath.Join(cjvHome, "toolchains", "lts-1.0.5"))
		for _, tool := range proxy.AllProxyTools() {
			assert.FileExists(t, filepath.Join(cjvHome, "bin", tool+".exe"))
		}
	}
}
