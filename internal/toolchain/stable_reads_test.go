package toolchain

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/fstx"
	"github.com/stretchr/testify/require"
)

func stableReadTestHome(t *testing.T) (string, *HomeLock) {
	t.Helper()
	home := t.TempDir()
	configtest.Isolate(t, home)
	require.NoError(t, config.EnsureDirs())
	require.NoError(t, os.Mkdir(filepath.Join(home, "toolchains", "custom"), 0o755))
	lock, err := LockHome(t.Context())
	require.NoError(t, err)
	t.Cleanup(func() { _ = lock.Close() })
	require.NoError(t, lock.Recover())
	require.NoError(t, lock.MigrateLegacy())
	return home, lock
}

func TestStableReadsAllowLookupButKeepOtherWritersExcluded(t *testing.T) {
	home, lock := stableReadTestHome(t)
	// Baseline: publication/recovery holds the lookup gate and excludes readers.
	ctx, cancel := context.WithTimeout(t.Context(), 40*time.Millisecond)
	defer cancel()
	_, _, err := PrepareActive(ctx, "custom", nil)
	require.ErrorIs(t, err, context.DeadlineExceeded)

	require.NoError(t, lock.WithStableReads(t.Context(), func() error {
		for _, selector := range []string{"custom", filepath.Join(home, "toolchains", "custom")} {
			readCtx, stop := context.WithTimeout(t.Context(), time.Second)
			dir, _, err := PrepareActive(readCtx, selector, nil)
			stop()
			if err != nil {
				return err
			}
			require.Equal(t, filepath.Join(home, "toolchains", "custom"), dir)
		}
		writeCtx, stop := context.WithTimeout(t.Context(), 40*time.Millisecond)
		defer stop()
		_, err := LockHome(writeCtx)
		require.ErrorIs(t, err, context.DeadlineExceeded, "the global writer lock protects snapshot consistency")
		return nil
	}))
	require.False(t, readablePhase(home), "publication cannot inherit the readable marker")
	ctx, stop := context.WithTimeout(t.Context(), 40*time.Millisecond)
	defer stop()
	_, _, err = PrepareActive(ctx, "custom", nil)
	require.ErrorIs(t, err, context.DeadlineExceeded, "leaving stable work must restore publication exclusivity")
}

func TestStableReadCancellationWaitsForReadersBeforeReturningToPublisher(t *testing.T) {
	home, lock := stableReadTestHome(t)
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	started, finishWork := make(chan struct{}), make(chan struct{})
	done := make(chan error, 1)
	go func() {
		done <- lock.WithStableReads(ctx, func() error {
			close(started)
			<-finishWork
			return ctx.Err()
		})
	}()
	<-started
	writer, lookup, err := lockActiveLookup(t.Context())
	require.NoError(t, err)
	require.Nil(t, writer)
	require.NotNil(t, lookup)
	defer lookup.Close() //nolint:errcheck
	cancel()
	close(finishWork)
	select {
	case err := <-done:
		t.Fatalf("writer returned while an active lookup still held its gate: %v", err)
	case <-time.After(40 * time.Millisecond):
	}
	require.NoError(t, lookup.Close())
	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled)
	case <-time.After(time.Second):
		t.Fatal("canceled preparation failed to reacquire its gate")
	}
	require.False(t, readablePhase(home))
	require.NoError(t, lock.Close())
	_, _, err = PrepareActive(t.Context(), "custom", nil)
	require.NoError(t, err)
}

func TestStableReadsRequireSuccessfulRecovery(t *testing.T) {
	configtest.Isolate(t, t.TempDir())
	lock, err := LockHome(t.Context())
	require.NoError(t, err)
	defer lock.Close() //nolint:errcheck
	err = lock.WithStableReads(t.Context(), func() error {
		t.Fatal("unrecovered state must not be advertised to readers")
		return nil
	})
	require.ErrorContains(t, err, "recovered home lock")
}

func TestPrepareActiveRechecksGateAfterAutomaticInstallation(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
	defer cancel()
	var publication *HomeLock
	defer func() {
		if publication != nil {
			_ = publication.Close()
		}
	}()
	_, _, err := PrepareActive(ctx, "lts-1.0.5", func(ctx context.Context) error {
		var err error
		publication, err = LockHome(ctx)
		if err != nil {
			return err
		}
		// A concurrent publisher has obtained home immediately after installation.
		// The installed directory exists, but final lookup must wait for its gate.
		return os.MkdirAll(filepath.Join(home, config.ToolchainsSubdir, "lts-1.0.5"), 0o755)
	})
	require.ErrorIs(t, err, context.DeadlineExceeded)
}

func TestStableReadWriterCrashNeverSkipsRecovery(t *testing.T) {
	const crashEnv = "CJV_TEST_STABLE_READ_CRASH"
	const snapshotID = "20261026T000000.000000000Z-aabbccddeeff"
	if mode := os.Getenv(crashEnv); mode != "" {
		home := os.Getenv(config.EnvHome)
		configtest.Isolate(t, home)
		lock, err := LockHome(t.Context())
		require.NoError(t, err)
		require.NoError(t, lock.Recover())
		require.NoError(t, lock.MigrateLegacy())
		require.NoError(t, lock.WithStableReads(t.Context(), func() error {
			staging := filepath.Join(home, config.HistorySubdir, "sts", snapshotID+config.StagingSuffix)
			require.NoError(t, os.MkdirAll(staging, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(staging, "partial"), []byte("history"), 0o644))
			if mode == "preparation" {
				os.Exit(73)
			}
			return nil
		}))
		// Publication starts only after the readable marker has been cleared.
		tx, err := fstx.NewToolchainTransaction(home, "custom")
		require.NoError(t, err)
		require.NoError(t, tx.RemoveDir(filepath.Join(home, config.ToolchainsSubdir, "custom")))
		os.Exit(74)
	}
	for _, mode := range []string{"preparation", "publication"} {
		t.Run(mode, func(t *testing.T) {
			home, lock := stableReadTestHome(t)
			require.NoError(t, lock.Close())
			cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestStableReadWriterCrashNeverSkipsRecovery$")
			cmd.Env = append(os.Environ(), crashEnv+"="+mode)
			output, err := cmd.CombinedOutput()
			var exit *exec.ExitError
			require.ErrorAs(t, err, &exit, "%s", output)
			code := 73
			if mode == "publication" {
				code = 74
			}
			require.Equal(t, code, exit.ExitCode(), "%s", output)
			require.Equal(t, mode == "preparation", readablePhase(home))
			ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
			defer cancel()
			dir, _, err := PrepareActive(ctx, "custom", nil)
			require.NoError(t, err)
			require.Equal(t, filepath.Join(home, config.ToolchainsSubdir, "custom"), dir)
			require.DirExists(t, dir)
			require.False(t, readablePhase(home))
			_, err = os.Stat(filepath.Join(home, config.HistorySubdir, "sts", snapshotID+config.StagingSuffix))
			require.True(t, errors.Is(err, os.ErrNotExist), "stale read marker must not prevent history cleanup: %v", err)
			journals, err := filepath.Glob(filepath.Join(home, config.ToolchainsSubdir, config.TxTempPrefix+"*"))
			require.NoError(t, err)
			require.Empty(t, journals)
		})
	}
}
