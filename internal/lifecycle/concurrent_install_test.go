package lifecycle_test

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Run real installations and recovery in separate processes. Both windows
// used to allow recovery (or another installer) to destroy a live install:
// a fully materialized staging tree, and an active journal after the swap.
func TestConcurrentInstallAndRecovery(t *testing.T) {
	if role := os.Getenv("CJV_TEST_CONCURRENT_ROLE"); role != "" {
		home := os.Getenv(config.EnvHome)
		config.IsolateForTest(t, home)
		if role == "owner" {
			pause := func() error {
				require.NoError(t, os.WriteFile(filepath.Join(home, "owner-ready"), nil, 0o600))
				require.Eventually(t, func() bool {
					_, err := os.Stat(filepath.Join(home, "release-owner"))
					return err == nil
				}, 20*time.Second, 10*time.Millisecond)
				return nil
			}
			if os.Getenv("CJV_TEST_CONCURRENT_STAGE") == "staging" {
				lifecycle.SetAfterStagingHook(t, pause)
			} else {
				lifecycle.SetAfterFinalizeHook(t, pause)
			}
		} else {
			require.NoError(t, os.WriteFile(filepath.Join(home, "contender-ready"), nil, 0o600))
		}
		if role == "recover" {
			require.NoError(t, toolchain.RecoverHome())
		} else {
			name := "owner-sdk"
			if role == "install" {
				name = "other-sdk"
			}
			require.NoError(t, lifecycle.InstallToolchainFromURL(t.Context(), name, os.Getenv("CJV_TEST_CONCURRENT_URL"), "", false, true, lifecycle.Options{}))
		}
		return
	}

	for _, stage := range []string{"staging", "transaction"} {
		for _, contender := range []string{"recover", "install"} {
			t.Run(stage+"/"+contender, func(t *testing.T) {
				home, _, serverURL := prepareInstallTest(t)
				ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
				defer cancel()
				start := func(role string) (<-chan error, *bytes.Buffer) {
					cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestConcurrentInstallAndRecovery$")
					cmd.Env = append(os.Environ(),
						"CJV_TEST_CONCURRENT_ROLE="+role,
						"CJV_TEST_CONCURRENT_STAGE="+stage,
						"CJV_TEST_CONCURRENT_URL="+serverURL+"/download/cangjie-sdk-1.0.5.zip")
					output := new(bytes.Buffer)
					cmd.Stdout, cmd.Stderr = output, output
					require.NoError(t, cmd.Start())
					done := make(chan error, 1)
					go func() { done <- cmd.Wait() }()
					return done, output
				}
				waitFor := func(name string) {
					require.Eventually(t, func() bool {
						_, err := os.Stat(filepath.Join(home, name))
						return err == nil
					}, 10*time.Second, 10*time.Millisecond)
				}
				ownerDone, ownerOutput := start("owner")
				waitFor("owner-ready")
				dest := filepath.Join(home, "toolchains", "owner-sdk")
				if stage == "staging" {
					assert.FileExists(t, compilerPath(config.StagingDir(dest)))
				} else {
					assert.FileExists(t, compilerPath(dest))
					journals, err := filepath.Glob(filepath.Join(home, "toolchains", ".fstx-*"))
					require.NoError(t, err)
					require.Len(t, journals, 1)
				}
				contenderDone, contenderOutput := start(contender)
				waitFor("contender-ready")
				select {
				case err := <-contenderDone:
					t.Fatalf("contender passed the live install lock: %v\n%s", err, contenderOutput.String())
				case <-time.After(200 * time.Millisecond):
				}
				require.NoError(t, os.WriteFile(filepath.Join(home, "release-owner"), nil, 0o600))
				err := <-ownerDone
				require.NoError(t, err, "%s", ownerOutput.String())
				err = <-contenderDone
				require.NoError(t, err, "%s", contenderOutput.String())
				assert.FileExists(t, compilerPath(dest))
				assert.NoDirExists(t, config.StagingDir(dest))
				journals, err := filepath.Glob(filepath.Join(home, "toolchains", ".fstx-*"))
				require.NoError(t, err)
				assert.Empty(t, journals)
				if contender == "install" {
					assert.FileExists(t, compilerPath(filepath.Join(home, "toolchains", "other-sdk")))
				}
			})
		}
	}
}
