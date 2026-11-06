package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/progress"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTrackingMutationPreservesMissingRecord(t *testing.T) {
	for _, operation := range []string{"update", "add-target", "update-target"} {
		t.Run(operation, func(t *testing.T) {
			f := newHistoryUpdateFixture(t)
			require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts", Targets: []string{"ohos"}}, Options{}))
			identity := "sts"
			if operation == "add-target" {
				identity += "-" + f.host + "-ohos"
			}
			dir := filepath.Join(f.home, "toolchains", identity)
			marker := filepath.Join(dir, "keep-user-content")
			require.NoError(t, os.WriteFile(marker, []byte("must survive"), 0o644))
			require.NoError(t, os.Remove(filepath.Join(dir, ".cjv", "toolchain.toml")))
			for attempt := 0; attempt < 2; attempt++ {
				var err error
				switch operation {
				case "update":
					_, err = UpdateInstalled(t.Context(), toolchain.ToolchainName{Channel: toolchain.STS}, Options{})
				case "update-target":
					_, err = UpdateInstalled(t.Context(), toolchain.ToolchainName{Channel: toolchain.STS, Target: f.host + "-ohos"}, Options{})
				default:
					err = Install(t.Context(), InstallRequest{Toolchain: "sts", NoUpdate: true, Targets: []string{"ohos"}}, Options{})
				}
				assert.Error(t, err, "a present SDK with missing metadata is damaged, not absent (attempt %d)", attempt)
				data, readErr := os.ReadFile(marker)
				require.NoError(t, readErr)
				require.Equal(t, "must survive", string(data))
			}
		})
	}
}

func TestPublicationRejectsDamagedRootAppearingDuringPreparation(t *testing.T) {
	for _, operation := range []string{"host", "target"} {
		t.Run(operation, func(t *testing.T) {
			f := newHistoryUpdateFixture(t)
			name := "sts"
			if operation == "target" {
				name += "-" + f.host + "-ohos"
			}
			marker := filepath.Join(f.home, "toolchains", "sts", "keep-user-content")
			changed := false
			opts := Options{Progress: targetInstallSink(func(e progress.Event) {
				if changed || e.Kind != progress.Extracting {
					return
				}
				// Deterministically interleave another writer while preparation has
				// released the home lock, before publication revalidates the snapshot.
				lock, err := toolchain.LockHome(t.Context())
				require.NoError(t, err)
				require.NoError(t, os.MkdirAll(filepath.Dir(marker), 0o755))
				require.NoError(t, os.WriteFile(marker, []byte("concurrent content"), 0o644))
				require.NoError(t, lock.Close())
				changed = true
			})}
			err := Install(t.Context(), InstallRequest{Toolchain: name}, opts)
			require.True(t, changed)
			require.ErrorContains(t, err, "changed during")
			// A retry must still reject the damaged root, rather than overwrite it.
			require.Error(t, Install(t.Context(), InstallRequest{Toolchain: name}, Options{}))
			data, err := os.ReadFile(marker)
			require.NoError(t, err)
			require.Equal(t, "concurrent content", string(data))
			if operation == "target" {
				require.NoDirExists(t, filepath.Join(f.home, "toolchains", name))
			}
		})
	}
}
