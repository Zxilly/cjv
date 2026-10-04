package lifecycle

import (
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestChannelGroupRetainsEverySDKWhenTargetIsUnavailable(t *testing.T) {
	for _, operation := range []string{"install", "update", "all"} {
		t.Run(operation, func(t *testing.T) {
			home := updateHome(t)
			selectRelease(t, "1.0.0", "ohos")
			require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts", Targets: []string{"ohos"}}, quietLifecycleOptions()))
			tuple, err := sdktarget.CurrentTargetTuple("", "ohos")
			require.NoError(t, err)
			selectRelease(t, "2.0.0")
			switch operation {
			case "install":
				err = Install(t.Context(), InstallRequest{Toolchain: "sts"}, quietLifecycleOptions())
			case "update":
				_, err = UpdateInstalled(t.Context(), parse(t, "sts"), quietLifecycleOptions())
			case "all":
				_, err = UpdateAll(t.Context(), quietLifecycleOptions())
			}
			require.Error(t, err)
			require.Equal(t, "sts-1.0.0", readRelease(t, home, "sts").Release)
			require.Equal(t, "sts-1.0.0-"+tuple, readRelease(t, home, "sts-"+tuple).Release)
		})
	}
}
