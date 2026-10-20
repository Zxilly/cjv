//go:build windows

package testutil

import (
	"context"
	"os"
	"os/exec"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sys/windows/registry"
)

func TestRegistryPathGuardSnapshotsAndReadsPath(t *testing.T) {
	guard, err := SaveRegistryPath()
	require.NoError(t, err)
	require.NotNil(t, guard)
	t.Cleanup(guard.Restore)
	assert.NotEmpty(t, guard.name)

	_, err = ReadRegistryPath()
	require.NoError(t, err)
}

func TestRegistryPathGuardSerializesProcesses(t *testing.T) {
	if mode := os.Getenv("CJV_TEST_REGISTRY_LOCK"); mode != "" {
		timeout := 2 * time.Minute
		if mode == "blocked" {
			timeout = 200 * time.Millisecond
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		guard, err := saveRegistryPath(ctx)
		if mode == "blocked" {
			if guard != nil {
				guard.Restore()
			}
			require.ErrorIs(t, err, context.DeadlineExceeded)
		} else {
			require.NoError(t, err)
			guard.Restore()
		}
		return
	}
	guard, err := SaveRegistryPath()
	require.NoError(t, err)
	t.Cleanup(guard.Restore)
	for _, mode := range []string{"blocked", "released"} {
		if mode == "released" {
			guard.Restore()
		}
		cmd := exec.Command(os.Args[0], "-test.run=^TestRegistryPathGuardSerializesProcesses$")
		cmd.Env = append(os.Environ(), "CJV_TEST_REGISTRY_LOCK="+mode)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "%s: %s", mode, out)
	}
}

func TestFindRegistryPathNameUsesStoredCasing(t *testing.T) {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\cjv-test-registry-guard`, registry.ALL_ACCESS)
	require.NoError(t, err)
	defer key.Close() //nolint:errcheck
	t.Cleanup(func() {
		_ = registry.DeleteKey(registry.CURRENT_USER, `Software\cjv-test-registry-guard`)
	})
	require.NoError(t, key.SetStringValue("PATH", "value"))

	assert.Equal(t, "PATH", findRegistryPathName(key))
}
