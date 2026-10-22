package testutil

import (
	"archive/zip"
	"bytes"
	"io/fs"
	"runtime"
	"testing"

	"github.com/Zxilly/cjv/internal/sdktools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMockSDKZipToolPermissions(t *testing.T) {
	data, _ := MockSDKZip()
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)

	// Inspect archive metadata on every host, including Windows where the
	// extracted file's permissions do not expose missing Unix execute bits.
	modes := make(map[string]fs.FileMode, len(reader.File))
	for _, file := range reader.File {
		modes[file.Name] = file.Mode()
	}
	for _, tool := range sdktools.AllProxyTools() {
		name := "cangjie/" + sdktools.ToolRelativePath(tool)
		if runtime.GOOS == "windows" {
			name += ".exe"
		}
		mode, ok := modes[name]
		require.True(t, ok, "archive must contain %s", name)
		assert.Equal(t, fs.FileMode(0o755), mode.Perm(), "%s must be executable after extraction", name)
	}
	for _, name := range []string{"cangjie/envsetup.sh", "cangjie/envsetup.ps1"} {
		mode, ok := modes[name]
		require.True(t, ok, "archive must contain %s", name)
		assert.Equal(t, fs.FileMode(0o666), mode.Perm(), "%s permissions must remain unchanged", name)
	}
}
