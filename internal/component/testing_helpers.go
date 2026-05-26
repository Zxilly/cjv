package component

import "testing"

// SetStdxReleaseBaseForTest overrides the LTS/STS stdx release base URL for the
// duration of the test, pointing ResolveAssetURL at a mock server, and restores
// the previous value via t.Cleanup. Used by higher-level packages (e.g. cli) to
// exercise the end-to-end component install flow against a local server.
func SetStdxReleaseBaseForTest(t *testing.T, base string) {
	t.Helper()
	prev := stdxReleaseBaseOverride
	stdxReleaseBaseOverride = base
	t.Cleanup(func() { stdxReleaseBaseOverride = prev })
}
