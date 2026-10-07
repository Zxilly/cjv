package cli

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/require"
)

func TestListRemoteDoesNotWaitForLocalPublication(t *testing.T) {
	for _, allPlatforms := range []bool{false, true} {
		t.Run(map[bool]string{false: "host", true: "all-platforms"}[allPlatforms], func(t *testing.T) {
			setupListRemote(t)
			lock, err := toolchain.LockHome(t.Context())
			require.NoError(t, err)
			defer lock.Close() //nolint:errcheck
			app := newApplication("dev", "")
			app.toolchainListRemoteChannel = "lts"
			app.toolchainListRemoteAllPlatforms = allPlatforms
			cmd, output := newListRemoteCmd()
			result := make(chan error, 1)
			go func() { result <- app.runToolchainListRemote(cmd, nil) }()
			select {
			case err := <-result:
				require.NoError(t, err)
				require.Contains(t, output.String(), "1.0.5")
			case <-time.After(5 * time.Second):
				_ = lock.Close()
				<-result
				t.Fatal("remote catalog waited for local publication")
			}
		})
	}
}

func TestListRemoteDoesNotRecoverLocalInstallations(t *testing.T) {
	setupListRemote(t)
	home, err := config.Home()
	require.NoError(t, err)
	journal := filepath.Join(home, "toolchains", ".fstx-interrupted", "journal.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(journal), 0o755))
	require.NoError(t, os.WriteFile(journal, []byte("{"), 0o644))
	staging := filepath.Join(home, "toolchains", "sts.staging")
	require.NoError(t, os.MkdirAll(staging, 0o755))
	app := newApplication("dev", "")
	app.toolchainListRemoteChannel = "lts"
	cmd, output := newListRemoteCmd()
	require.NoError(t, app.runToolchainListRemote(cmd, nil))
	require.Contains(t, output.String(), "1.0.5")
	require.FileExists(t, journal)
	require.DirExists(t, staging)
}

func TestListRemoteDoesNotScanLocalInstallations(t *testing.T) {
	setupListRemote(t)
	home, err := config.Home()
	require.NoError(t, err)
	root := filepath.Join(home, "toolchains")
	require.NoError(t, os.Remove(root))
	require.NoError(t, os.WriteFile(root, []byte("not a directory"), 0o644))
	app := newApplication("dev", "")
	app.toolchainListRemoteChannel = "lts"
	cmd, output := newListRemoteCmd()
	require.NoError(t, app.runToolchainListRemote(cmd, nil))
	require.Contains(t, output.String(), "1.0.5")
	require.FileExists(t, root)
}

func TestListRemotePropagatesCancellation(t *testing.T) {
	for _, allPlatforms := range []bool{false, true} {
		t.Run(map[bool]string{false: "host", true: "all-platforms"}[allPlatforms], func(t *testing.T) {
			setupListRemote(t)
			app := newApplication("dev", "")
			app.toolchainListRemoteChannel = "lts"
			app.toolchainListRemoteAllPlatforms = allPlatforms
			cmd, output := newListRemoteCmd()
			ctx, cancel := context.WithCancel(t.Context())
			cancel()
			cmd.SetContext(ctx)
			require.ErrorIs(t, app.runToolchainListRemote(cmd, nil), context.Canceled)
			require.Empty(t, output.String())
		})
	}
}
