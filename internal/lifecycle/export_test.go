package lifecycle

import (
	"context"
	"testing"

	"github.com/Zxilly/cjv/internal/toolchain"
)

type InstallationDistribution = installationDistribution

func OpenInstallationDistribution(ctx context.Context, opts Options) (*InstallationDistribution, error) {
	return openInstallationDistribution(ctx, opts)
}

// UpgradeToolchain exposes the production group installer with a caller-chosen
// identity, so tests can exercise replacement without another channel lookup.
func UpgradeToolchain(ctx context.Context, identity string, rt ResolvedToolchain, d *InstallationDistribution, opts Options) (bool, error) {
	name, err := toolchain.ParseToolchainName(identity)
	if err != nil {
		return false, err
	}
	return installGroup(ctx, d, name, rt, InstallRequest{Toolchain: identity}, opts)
}

// SetAfterFinalizeHook installs hook to run after the managed binary and proxy
// links have been established for a newly placed toolchain, before the
// transaction commits. The hook is cleared when t finishes.
func SetAfterFinalizeHook(t *testing.T, hook func() error) {
	t.Helper()
	previous := afterFinalizeHook
	afterFinalizeHook = hook
	t.Cleanup(func() { afterFinalizeHook = previous })
}

// SetAfterPublishHook installs hook to run after the first default toolchain
// has been published (settings updated, PATH configured when requested) and
// before the transaction completes. The hook is cleared when t finishes.
func SetAfterPublishHook(t *testing.T, hook func() error) {
	t.Helper()
	previous := afterPublishHook
	afterPublishHook = hook
	t.Cleanup(func() { afterPublishHook = previous })
}

// SetAfterStagingHook pauses placement before the staging tree is published.
func SetAfterStagingHook(t *testing.T, hook func() error) {
	t.Helper()
	previous := afterStagingHook
	afterStagingHook = hook
	t.Cleanup(func() { afterStagingHook = previous })
}
