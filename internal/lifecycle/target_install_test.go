package lifecycle

import (
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/progress"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type targetInstallSink func(progress.Event)

func (f targetInstallSink) Report(e progress.Event) { f(e) }

func TestTargetPublicationRejectsChangedHost(t *testing.T) {
	for _, operation := range []string{"install", "update", "auto-install"} {
		t.Run(operation, func(t *testing.T) {
			home := updateHome(t)
			selectRelease(t, "1.0.0", "ohos")
			require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, quietLifecycleOptions()))
			tuple, err := sdktarget.CurrentTargetTuple("", "ohos")
			require.NoError(t, err)
			changed := false
			opts := Options{Progress: targetInstallSink(func(e progress.Event) {
				if e.Kind != progress.FetchingManifest || changed {
					return
				}
				changed = true
				// The old operation has selected its host but has not acquired the
				// install lock yet. Another operation completes the host upgrade.
				selectRelease(t, "2.0.0", "ohos")
				require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, quietLifecycleOptions()))
			})}
			switch operation {
			case "install":
				err = Install(t.Context(), InstallRequest{Toolchain: "sts-" + tuple}, opts)
			case "update":
				_, err = UpdateInstalled(t.Context(), parse(t, "sts-"+tuple), opts)
			case "auto-install":
				err = InstallTargetsForToolchain(t.Context(), "sts", []string{"ohos"}, opts)
			}
			require.ErrorContains(t, err, "changed during target installation")
			assert.True(t, changed)
			assert.Equal(t, "sts-2.0.0", readRelease(t, home, "sts").Release)
			assert.NoDirExists(t, filepath.Join(home, "toolchains", "sts-"+tuple))
		})
	}
}
