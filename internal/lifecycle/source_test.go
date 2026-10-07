package lifecycle

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Zxilly/cjv/internal/fstx"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/require"
)

func installationOperations() map[string]func(context.Context) error {
	return map[string]func(context.Context) error{
		"install": func(ctx context.Context) error {
			return Install(ctx, InstallRequest{Toolchain: "sts"}, Options{})
		},
		"update": func(ctx context.Context) error {
			_, err := UpdateInstalled(ctx, toolchain.ToolchainName{Channel: toolchain.STS}, Options{})
			return err
		},
		"update-all": func(ctx context.Context) error {
			_, err := UpdateAll(ctx, Options{})
			return err
		},
		"targets": func(ctx context.Context) error {
			return InstallTargetsForToolchain(ctx, "sts", []string{"ohos"}, Options{})
		},
	}
}

func TestInstallationDistributionWaitHonorsCancellation(t *testing.T) {
	for name, operation := range installationOperations() {
		t.Run(name, func(t *testing.T) {
			updateHome(t)
			lock, err := toolchain.LockHome(t.Context())
			require.NoError(t, err)
			defer lock.Close() //nolint:errcheck
			ctx, cancel := context.WithTimeout(t.Context(), 100*time.Millisecond)
			defer cancel()
			result := make(chan error, 1)
			go func() { result <- operation(ctx) }()
			select {
			case err := <-result:
				require.ErrorIs(t, err, context.DeadlineExceeded)
			case <-time.After(5 * time.Second):
				// Release the lock before waiting for a regressed implementation.
				_ = lock.Close()
				<-result
				t.Fatal("installation ignored cancellation while waiting for home")
			}
		})
	}
}

func TestInstallationDistributionRequiresSuccessfulRecovery(t *testing.T) {
	for name, operation := range installationOperations() {
		t.Run(name, func(t *testing.T) {
			home := updateHome(t)
			journal := filepath.Join(home, "toolchains", ".fstx-interrupted", "journal.json")
			require.NoError(t, os.MkdirAll(filepath.Dir(journal), 0o755))
			require.NoError(t, os.WriteFile(journal, []byte("{"), 0o644))
			var recoveryErr *fstx.RecoveryError
			require.ErrorAs(t, operation(t.Context()), &recoveryErr)
			require.FileExists(t, journal)
		})
	}
}

func TestUpdateAllRecoversBeforeCheckingEmptyHome(t *testing.T) {
	home := updateHome(t)
	selectRelease(t, "1.0.0")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, Options{}))
	dir := filepath.Join(home, "toolchains", "sts")
	require.NoError(t, os.Rename(dir, dir+".old"))
	selectRelease(t, "2.0.0")
	report, err := UpdateAll(t.Context(), Options{})
	require.NoError(t, err)
	require.False(t, report.NoneInstalled)
	require.Equal(t, "sts-2.0.0", readRelease(t, home, "sts").Release)
	require.NoDirExists(t, dir+".old")
}
