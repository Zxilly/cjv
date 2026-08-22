//go:build !windows

package selfupdate

import "testing"

func fileIsHidden(t *testing.T, _ string) bool {
	t.Helper()
	return false
}
