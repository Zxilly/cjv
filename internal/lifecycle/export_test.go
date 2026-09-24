package lifecycle

import "testing"

// UpgradeToolchain exposes the upgrade step with a caller-chosen current
// name, so tests can pin which installed version is replaced independently
// of the channel lookup UpdateInstalled performs.
var UpgradeToolchain = upgradeToolchain

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
