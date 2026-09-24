package selfupdate

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateSkipsNetworkForUnsupportedOrDevBuilds(t *testing.T) {
	result, err := Update(context.Background(), "", "1.0.0")
	require.NoError(t, err)
	assert.Equal(t, Result{CurrentVersion: "1.0.0", Version: "1.0.0", Status: StatusSkipped}, result)
	result, err = Update(context.Background(), "https://example.invalid/owner/repo/releases", "dev")
	require.NoError(t, err)
	assert.Equal(t, Result{CurrentVersion: "dev", Version: "dev", Status: StatusDevelopment}, result)
}

// Exercise the same Update interface used by managed updates, including the
// build-selected release discovery adapter, checksums and real file replacement.
func TestUpdateReportsInstalledRelease(t *testing.T) {
	for _, tc := range []struct {
		name, tag, version string
		status             Status
		invalidChecksum    bool
	}{
		{name: "already current", tag: "v1.0.0", version: "1.0.0", status: StatusUpToDate},
		{name: "updated", tag: "v1.1.0", version: "1.1.0", status: StatusUpdated},
		{name: "checksum mismatch", tag: "v1.1.0", invalidChecksum: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			config.IsolateForTest(t, home)
			t.Setenv(config.EnvMaxRetries, "0")
			managed := filepath.Join(home, "bin", sdktools.CjvBinaryName())
			require.NoError(t, os.MkdirAll(filepath.Dir(managed), 0o755))
			require.NoError(t, os.WriteFile(managed, []byte("old executable"), 0o755))
			archive := zipExecutable(t, platformBinaryName(updateTestBinaryName(), runtime.GOOS), []byte("new executable"))
			digest := fmt.Sprintf("%x", sha256.Sum256(archive))
			if tc.invalidChecksum {
				digest = strings.Repeat("0", 64)
			}
			updateURL := serveUpdateRelease(t, tc.tag, archive, digest)
			result, err := Update(context.Background(), updateURL, "1.0.0")
			if tc.invalidChecksum {
				require.Error(t, err)
				assert.NotEqual(t, StatusUpdated, result.Status)
			} else {
				require.NoError(t, err)
				assert.Equal(t, Result{CurrentVersion: "1.0.0", Version: tc.version, Status: tc.status}, result)
			}
			installed, err := os.ReadFile(managed)
			require.NoError(t, err)
			if tc.status == StatusUpdated {
				assert.Equal(t, "new executable", string(installed))
			} else {
				assert.Equal(t, "old executable", string(installed))
			}
		})
	}
}
