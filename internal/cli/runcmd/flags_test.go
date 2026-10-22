package runcmd

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/cli/clitest"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/resolve"
	"github.com/Zxilly/cjv/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type forwardedInvocation struct {
	Args        []string `json:"args"`
	Config      string   `json:"config"`
	Offline     string   `json:"offline"`
	AutoInstall string   `json:"auto_install"`
}

func TestForwardedInvocationHelper(t *testing.T) {
	path := os.Getenv("CJV_TEST_INVOCATION_REPORT")
	if path == "" {
		return
	}
	var args []string
	for i, arg := range os.Args {
		if arg == "--" {
			args = os.Args[i+1:]
			break
		}
	}
	data, err := json.Marshal(forwardedInvocation{Args: args, Config: os.Getenv(config.EnvConfigFile), Offline: os.Getenv("CJV_OFFLINE"), AutoInstall: os.Getenv(config.EnvAutoInstall)})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o600))
}

func TestRunExecGlobalFlagsAndChildArguments(t *testing.T) {
	for _, command := range []string{"run", "exec"} {
		for _, before := range []bool{false, true} {
			t.Run(command+map[bool]string{false: "/local-prefix", true: "/root-prefix"}[before], func(t *testing.T) {
				home := t.TempDir()
				configtest.Isolate(t, home)
				t.Setenv(config.EnvAutoInstall, "1")
				t.Setenv("CJV_OFFLINE", "0")
				t.Setenv(config.EnvLog, "")
				root := clitest.Root(Register)
				var stderr bytes.Buffer
				root.Root.SetErr(&stderr)
				configPath := filepath.Join(t.TempDir(), "selected.toml")
				require.NoError(t, os.WriteFile(configPath, []byte("default_toolchain = 'test-sdk'\nauto_install = true\n"), 0o600))
				require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "test-sdk"), 0o755))
				report := filepath.Join(t.TempDir(), "child.json")
				t.Setenv("CJV_TEST_INVOCATION_REPORT", report)
				flags := []string{"--quiet", "--offline", "--no-auto-install", "--config", configPath}
				args := []string{command}
				if before {
					args = append(append([]string{}, flags...), args...)
				} else {
					args = append(args, flags...)
				}
				if command == "run" {
					args = append(args, "test-sdk")
				}
				childArgs := []string{"--quiet", "--verbose", "--json", "--config=child.toml", "--offline=false", "--no-auto-install=false", "+child-selector"}
				args = append(args, os.Args[0], "-test.run=^TestForwardedInvocationHelper$", "--")
				args = append(args, childArgs...)
				_, err := clitest.Execute(t, root, args...)
				require.NoError(t, err)
				require.True(t, root.Output.Quiet)
				require.Empty(t, stderr.String())
				var got forwardedInvocation
				data, err := os.ReadFile(report)
				require.NoError(t, err)
				require.NoError(t, json.Unmarshal(data, &got))
				assert.Equal(t, childArgs, got.Args)
				assert.Equal(t, configPath, got.Config)
				assert.Equal(t, "1", got.Offline)
				assert.Equal(t, "0", got.AutoInstall)
				assert.Empty(t, os.Getenv(config.EnvConfigFile))
				assert.Equal(t, "0", os.Getenv("CJV_OFFLINE"))
				assert.Equal(t, "1", os.Getenv(config.EnvAutoInstall))
				stored, err := config.LoadSettings(configPath)
				require.NoError(t, err)
				assert.True(t, stored.AutoInstall, "invocation policy must not persist")
			})
		}
	}
}

func TestExecQuietSuppressesAutomaticInstallationProgress(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	server := testutil.ValidMockServer(t)
	sf, err := config.DefaultSettingsFile()
	require.NoError(t, err)
	url := server.URL + "/sdk-versions.json"
	name := "lts"
	_, err = sf.Update(config.SettingsUpdate{ManifestURL: &url, DefaultToolchain: &name})
	require.NoError(t, err)
	root := clitest.Root(Register)
	var stderr bytes.Buffer
	root.Root.SetErr(&stderr)
	_, err = clitest.Execute(t, root, "exec", "--quiet", "go", "version")
	require.NoError(t, err)
	assert.Empty(t, stderr.String())
	require.DirExists(t, filepath.Join(home, "toolchains", "lts"))
}

func TestWhichAndEnvsetupQuietAutomaticInstallation(t *testing.T) {
	for _, command := range []string{"which", "envsetup"} {
		t.Run(command, func(t *testing.T) {
			home := t.TempDir()
			configtest.Isolate(t, home)
			server := testutil.ValidMockServer(t)
			sf, err := config.DefaultSettingsFile()
			require.NoError(t, err)
			url, name := server.URL+"/sdk-versions.json", "lts"
			_, err = sf.Update(config.SettingsUpdate{ManifestURL: &url, DefaultToolchain: &name})
			require.NoError(t, err)
			root := clitest.Root(Register)
			var stderr bytes.Buffer
			root.Root.SetErr(&stderr)
			args := []string{command, "--quiet"}
			if command == "envsetup" {
				args = append(args, "--shell=bash")
			}
			output, err := clitest.Execute(t, root, args...)
			require.NoError(t, err)
			assert.NotEmpty(t, output)
			assert.Empty(t, stderr.String())
			require.DirExists(t, filepath.Join(home, "toolchains", "lts"))
		})
	}
}

func TestRunExecVerboseConflictAndFlagErrors(t *testing.T) {
	for _, command := range []string{"run", "exec"} {
		t.Run(command, func(t *testing.T) {
			t.Setenv(config.EnvLog, "")
			logger := slog.Default()
			root := clitest.Root(Register)
			_, err := clitest.Execute(t, root, command, "--quiet", "--verbose")
			require.ErrorContains(t, err, "--quiet and --verbose")
			assert.Same(t, logger, slog.Default())
			root = clitest.Root(Register)
			_, err = clitest.Execute(t, root, command, "--config")
			require.ErrorContains(t, err, "--config")
			root = clitest.Root(Register)
			_, err = clitest.Execute(t, root, command, "--quiet=invalid")
			require.ErrorContains(t, err, "invalid --quiet")
		})
	}
}

func TestExecNoAutoInstallRejectsMissingWithoutCallingInstaller(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	t.Setenv(config.EnvToolchain, "lts-1.0.5")
	previous := resolve.AutoInstallFunc
	t.Cleanup(func() { resolve.AutoInstallFunc = previous })
	resolve.AutoInstallFunc = func(_ context.Context, _ string, _ []string) error {
		t.Fatal("automatic installation must be disabled")
		return nil
	}
	_, err := execute(t, "exec", "--no-auto-install", "go", "version")
	require.Error(t, err)
	assert.NoDirExists(t, filepath.Join(home, "toolchains", "lts-1.0.5"))
}

func TestRunExecApplyFinalFlagValuesOnce(t *testing.T) {
	for _, command := range []string{"run", "exec"} {
		for _, invalidEnvironment := range []bool{false, true} {
			t.Run(command+map[bool]string{false: "/last-value", true: "/invalid-env-override"}[invalidEnvironment], func(t *testing.T) {
				home := t.TempDir()
				configtest.Isolate(t, home)
				t.Setenv(config.EnvToolchain, "test-sdk")
				t.Setenv("CJV_OFFLINE", "0")
				t.Setenv(config.EnvAutoInstall, "1")
				if invalidEnvironment {
					t.Setenv(config.EnvAutoInstall, "invalid")
				}
				require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", "test-sdk"), 0o755))
				report := filepath.Join(t.TempDir(), "child.json")
				t.Setenv("CJV_TEST_INVOCATION_REPORT", report)
				args := []string{"--offline", "--no-auto-install", command, "--offline=false", "--no-auto-install=false"}
				if invalidEnvironment {
					args = []string{command, "--no-auto-install"}
				}
				if command == "run" {
					args = append(args, "test-sdk")
				}
				args = append(args, os.Args[0], "-test.run=^TestForwardedInvocationHelper$", "--")
				_, err := clitest.Execute(t, clitest.Root(Register), args...)
				require.NoError(t, err)
				data, err := os.ReadFile(report)
				require.NoError(t, err)
				var got forwardedInvocation
				require.NoError(t, json.Unmarshal(data, &got))
				assert.Equal(t, "0", got.Offline)
				if invalidEnvironment {
					assert.Equal(t, "0", got.AutoInstall)
					assert.Equal(t, "invalid", os.Getenv(config.EnvAutoInstall))
				} else {
					assert.Equal(t, "1", got.AutoInstall)
					assert.Equal(t, "1", os.Getenv(config.EnvAutoInstall))
				}
			})
		}
	}
}
