//go:build !linux

package fsops

import "testing"

// crossDeviceDir skips: no second filesystem is known on this platform.
func crossDeviceDir(t *testing.T) string {
	t.Helper()
	t.Skip("no known second filesystem on this platform")
	return ""
}
