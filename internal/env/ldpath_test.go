package env

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// EnsureLibraryPath is a no-op on Windows (DLLs are found via PATH).
// On Unix it sets LD_LIBRARY_PATH / DYLD_FALLBACK_LIBRARY_PATH.

func TestEnsureLibraryPath_NoOp(t *testing.T) {
	cfg := NewEnvConfig()
	assert.NotPanics(t, func() {
		EnsureLibraryPath(cfg, "/some/sdk")
	})
}
