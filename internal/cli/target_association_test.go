package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/resolve"
	"github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/require"
)

// Exercise each production consumer against the same installed state. A host
// selector, its directory identity and the artifact release need not coincide.
func TestCrossSDKAssociationAcrossCommands(t *testing.T) {
	for _, test := range []struct {
		name     string
		pinned   bool
		alias    bool
		other    bool
		legacy   bool
		mismatch bool
	}{
		{name: "tracking"},
		{name: "pinned", pinned: true},
		{name: "explicit alias", alias: true},
		{name: "different host", other: true},
		{name: "legacy tuple fallback", pinned: true, legacy: true},
		{name: "stale target", mismatch: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			home := t.TempDir()
			config.IsolateForTest(t, home)
			t.Setenv(config.EnvToolchain, "")
			require.NoError(t, toolchain.RecoverHome())
			current, err := target.CurrentHostTuple("")
			require.NoError(t, err)
			other := "linux-x64"
			if current == other {
				other = "win32-x64"
			}
			hostTuple, decoyTuple := current, other
			hostName := "sts"
			if test.pinned {
				hostName += "-2.0.0"
			}
			if test.other {
				hostTuple, decoyTuple = other, current
				hostName += "-" + other
			}
			selector := hostName
			if test.alias {
				selector += "-" + hostTuple
			}
			hostRelease := "sts-2.0.0"
			if test.other {
				hostRelease += "-" + other
			}
			hostDir := filepath.Join(home, "toolchains", hostName)
			require.NoError(t, os.MkdirAll(hostDir, 0755))
			if !test.legacy {
				require.NoError(t, toolchain.WriteInstallation(hostDir, toolchain.Installation{Release: hostRelease, Tuple: hostTuple}))
			}
			prefix := "sts"
			if test.pinned {
				prefix += "-2.0.0"
			}
			targetName := prefix + "-" + hostTuple + "-ohos"
			targetDir := filepath.Join(home, "toolchains", targetName)
			version := "2.0.0"
			if test.mismatch {
				version = "1.0.0"
			}
			require.NoError(t, toolchain.WriteInstallation(targetDir, toolchain.Installation{Release: "sts-" + version + "-" + hostTuple + "-ohos", Tuple: hostTuple + "-ohos"}))
			decoyDir := filepath.Join(home, "toolchains", prefix+"-"+decoyTuple+"-ohos")
			require.NoError(t, toolchain.WriteInstallation(decoyDir, toolchain.Installation{Release: "sts-2.0.0-" + decoyTuple + "-ohos", Tuple: decoyTuple + "-ohos"}))
			peerPrefix := "sts-2.0.0"
			if test.pinned {
				peerPrefix = "sts"
			}
			peerDir := filepath.Join(home, "toolchains", peerPrefix+"-"+hostTuple+"-ohos")
			require.NoError(t, toolchain.WriteInstallation(peerDir, toolchain.Installation{Release: "sts-2.0.0-" + hostTuple + "-ohos", Tuple: hostTuple + "-ohos"}))

			text, err := executeWithOutput(t, newApplication("dev", ""), []string{"target", "list", "--toolchain", selector, "--installed", "--json"})
			require.NoError(t, err)
			var listed targetListResult
			require.NoError(t, json.Unmarshal([]byte(text), &listed))
			if test.mismatch {
				require.Empty(t, listed.Targets)
			} else {
				require.Equal(t, []targetEntry{{Name: "ohos", Installed: true}}, listed.Targets)
			}
			text, componentErr := executeWithOutput(t, newApplication("dev", ""), []string{"component", "list", "--toolchain", selector, "--target", "ohos", "--installed", "--json"})
			active, activeErr := resolve.ActiveTarget(t.Context(), selector, "ohos")
			if test.mismatch {
				var missing *cjverr.ToolchainNotInstalledError
				require.ErrorAs(t, componentErr, &missing)
				require.Equal(t, targetName, missing.Name)
				require.ErrorAs(t, activeErr, &missing)
			} else {
				require.NoError(t, componentErr)
				var components componentListResult
				require.NoError(t, json.Unmarshal([]byte(text), &components))
				require.Equal(t, targetName, components.Toolchain)
				require.NoError(t, activeErr)
				require.Equal(t, targetDir, active.Dir)
			}
			// Removal is by ownership, even for a stale target that cannot run.
			_, err = executeWithOutput(t, newApplication("dev", ""), []string{"target", "remove", "ohos", "--toolchain", selector, "--json"})
			require.NoError(t, err)
			require.NoDirExists(t, targetDir)
			require.DirExists(t, hostDir)
			require.DirExists(t, decoyDir)
			require.DirExists(t, peerDir)
		})
	}
}
