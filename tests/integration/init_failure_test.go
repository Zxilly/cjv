//go:build integration

package integration

import (
	"crypto/sha256"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Exercise the real process status through both the CLI and bootstrap scripts.
// The SDK has a valid checksum but is not a ZIP, so installation must fail.
func TestIntegrationInitFailureStatus(t *testing.T) {
	archive := []byte("invalid SDK archive")
	server := newMockSDKServer(t, archive, fmt.Sprintf("%x", sha256.Sum256(archive)))
	installers := []string{"cjv"}
	if runtime.GOOS == "windows" {
		for _, shell := range []string{"powershell", "pwsh"} {
			if _, err := exec.LookPath(shell); err == nil {
				installers = append(installers, shell)
			}
		}
	} else {
		installers = append(installers, "sh")
	}
	for _, installer := range installers {
		for _, toolchain := range []string{"lts", "none"} {
			t.Run(installer+"/"+toolchain, func(t *testing.T) {
				home := setupIntegrationEnvWithoutManagedBinary(t, server.URL)
				var stdout, stderr string
				var err error
				if installer == "cjv" {
					stdout, stderr, err = runCJVEnv(t, buildCJV(t), home,
						[]string{"CJV_NO_PATH_SETUP=1", "CJV_DIST_SERVER="},
						"init", "-y", "--no-modify-path", "--default-toolchain", toolchain)
				} else {
					var args []string
					if installer == "sh" {
						args = []string{filepath.Join("..", "..", "web", "public", "install.sh"),
							"-y", "--no-modify-path", "--default-toolchain", toolchain}
					} else {
						args = []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File",
							filepath.Join("..", "..", "web", "public", "install.ps1"),
							"-Yes", "-NoModifyPath", "-DefaultToolchain", toolchain}
					}
					cmd := exec.Command(installer, args...)
					cmd.Env = installScriptEnv(server.URL, home, "CJV_DIST_SERVER=", "CJV_MIRROR=", "CJV_VERSION=")
					var out, diagnostics strings.Builder
					cmd.Stdout, cmd.Stderr = &out, &diagnostics
					err = cmd.Run()
					stdout, stderr = out.String(), diagnostics.String()
				}
				if toolchain == "none" {
					require.NoError(t, err, "stdout=%s stderr=%s", stdout, stderr)
					assert.Contains(t, stdout, "cjv is installed now.")
				} else {
					require.Error(t, err, "stdout=%s stderr=%s", stdout, stderr)
					var exitErr *exec.ExitError
					require.ErrorAs(t, err, &exitErr)
					assert.Equal(t, 1, exitErr.ExitCode())
					assert.Contains(t, stderr, "Failed to install toolchain 'lts'")
					assert.Contains(t, stderr, "Fix the error above")
					assert.Contains(t, stderr, " install 'lts'")
					assert.Contains(t, stderr, " list-remote")
					assert.Contains(t, stderr, filepath.Join(home, "bin", sdktools.CjvBinaryName()))
					assert.NotContains(t, stdout, "cjv is installed now.")
					assert.NoDirExists(t, filepath.Join(home, "toolchains", "lts-1.0.5"))
				}
				assert.FileExists(t, filepath.Join(home, "bin", sdktools.CjvBinaryName()))
			})
		}
	}
}
