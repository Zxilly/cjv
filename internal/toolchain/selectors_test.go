package toolchain

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/target"
	"github.com/stretchr/testify/require"
)

func TestVersionSelectorsChooseInstalledConcreteRelease(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	for _, name := range []string{"sts-1.2.1", "sts-1.2.10", "sts-1.2.11-beta.1", "nightly-1.3.0-alpha.20261004001050", "nightly-1.3.0-alpha.20261005001050"} {
		require.NoError(t, os.MkdirAll(filepath.Join(home, "toolchains", name), 0755))
	}
	for _, test := range []struct{ input, actual string }{{"1.2", "sts-1.2.10"}, {"sts-1.2", "sts-1.2.10"}, {"nightly-2026-10-04", "nightly-1.3.0-alpha.20261004001050"}} {
		name, err := ParseToolchainName(test.input)
		require.NoError(t, err)
		dir, err := FindInstalled(name)
		require.NoError(t, err)
		require.Equal(t, test.actual, filepath.Base(dir))
	}
	require.False(t, MatchesVersionSelector("2026-02-30", "1.3.0-alpha.20260230000000"))
}

func TestExplicitHostSelectorsFindLegacyIdentity(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)
	host, err := target.CurrentHostTuple("")
	require.NoError(t, err)
	dir := filepath.Join(home, "toolchains", "sts-1.0.0")
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, WriteInstallation(dir, Installation{Release: "sts-1.0.0", Tuple: host}))
	for _, input := range []string{"sts-1.0.0-" + host, "sts-1.0-" + host, "1.0.0-" + host, "1.0-" + host} {
		name, err := ParseToolchainName(input)
		require.NoError(t, err)
		actual, err := FindInstalled(name)
		require.NoError(t, err)
		require.Equal(t, dir, actual)
	}
}
