package lifecycle

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/stretchr/testify/require"
)

func TestHostRemovalRemovesOnlyOwnedTargets(t *testing.T) {
	host, err := target.CurrentHostTuple("")
	require.NoError(t, err)
	for _, input := range []string{"sts", "sts-1.0.0", "1.0.0", "sts-1.0", "sts-" + host, "sts-1.0.0-" + host} {
		t.Run(input, func(t *testing.T) {
			home := updateHome(t)
			selectRelease(t, "1.0.0", "ohos")
			identity := "sts-1.0.0"
			otherIdentity := "sts"
			if input == "sts" || input == "sts-"+host {
				identity, otherIdentity = otherIdentity, identity
			}
			require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: identity, Targets: []string{"ohos"}}, Options{}))
			tuple, err := target.CurrentTargetTuple("", "ohos")
			require.NoError(t, err)
			owned := identity + "-" + tuple
			roots, err := component.RootsFor(owned)
			require.NoError(t, err)
			for _, dir := range []string{roots.DocsDir, roots.StdxDir} {
				require.NoError(t, os.MkdirAll(dir, 0o755))
			}
			otherTuple := "linux-x64-ohos"
			if tuple == otherTuple {
				otherTuple = "win32-x64-ohos"
			}
			retained := []string{otherIdentity + "-" + tuple, "sts-2.0.0-" + tuple, "lts-1.0.0-" + tuple, identity + "-" + otherTuple}
			for _, name := range retained {
				dir := filepath.Join(home, "toolchains", name)
				parsed := parse(t, name)
				if parsed.Version == "" {
					parsed.Version = "1.0.0"
				}
				require.NoError(t, os.MkdirAll(dir, 0o755))
				require.NoError(t, toolchain.WriteInstallation(dir, toolchain.Installation{Release: parsed.String(), Tuple: parsed.Target}))
			}
			require.NoError(t, RemoveToolchain(input))
			require.NoDirExists(t, filepath.Join(home, "toolchains", identity))
			for _, dir := range []string{roots.TcDir, roots.DocsDir, roots.StdxDir} {
				require.NoDirExists(t, dir)
			}
			for _, name := range retained {
				require.DirExists(t, filepath.Join(home, "toolchains", name))
			}
		})
	}
}

func TestHostGroupRemovalRestoresHostWhenLaterTargetIsBusy(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows directory handle blocks target retirement")
	}
	for _, identity := range []string{"sts", "sts-1.0.0"} {
		t.Run(identity, func(t *testing.T) {
			home := updateHome(t)
			selectRelease(t, "1.0.0", "ohos")
			require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: identity, Targets: []string{"ohos"}}, Options{}))
			other := "sts-1.0.0"
			if identity == other {
				other = "sts"
			}
			require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: other}, Options{}))
			tuple, err := target.CurrentTargetTuple("", "ohos")
			require.NoError(t, err)
			before := readChoices(t)
			t.Chdir(filepath.Join(home, "toolchains", identity+"-"+tuple))
			err = RemoveToolchain(identity)
			require.Error(t, err)
			require.Equal(t, "sts-1.0.0", readRelease(t, home, identity).Release)
			require.Equal(t, "sts-1.0.0-"+tuple, readRelease(t, home, identity+"-"+tuple).Release)
			require.Equal(t, before, readChoices(t))
			require.NoError(t, os.Chdir(home))
			require.NoError(t, RemoveToolchain(identity))
			require.NoDirExists(t, filepath.Join(home, "toolchains", identity))
			require.NoDirExists(t, filepath.Join(home, "toolchains", identity+"-"+tuple))
			require.Equal(t, "sts-1.0.0", readRelease(t, home, other).Release)
		})
	}
}

func TestHostRemovalIncludesUnusableOwnedTargets(t *testing.T) {
	for _, state := range []string{"stale", "damaged"} {
		t.Run(state, func(t *testing.T) {
			home := updateHome(t)
			selectRelease(t, "1.0.0", "ohos")
			require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts", Targets: []string{"ohos"}}, Options{}))
			tuple, err := target.CurrentTargetTuple("", "ohos")
			require.NoError(t, err)
			cross := filepath.Join(home, "toolchains", "sts-"+tuple)
			if state == "stale" {
				require.NoError(t, toolchain.WriteInstallation(cross, toolchain.Installation{Release: "sts-0.9.0-" + tuple, Tuple: tuple}))
			} else {
				require.NoError(t, os.WriteFile(filepath.Join(cross, ".cjv", "toolchain.toml"), []byte("broken = ["), 0o644))
			}
			require.NoError(t, RemoveToolchain("sts"))
			require.NoDirExists(t, cross)
		})
	}
}
