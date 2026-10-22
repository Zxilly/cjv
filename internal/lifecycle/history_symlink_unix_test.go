//go:build unix

package lifecycle

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/stretchr/testify/require"
)

func historyAliasedParent(t *testing.T) string {
	t.Helper()
	base := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(base, "real"), 0o755))
	alias := filepath.Join(base, "alias")
	require.NoError(t, os.Symlink("real", alias))
	return alias
}

func TestHistoryInternalLinksUnderSymlinkedParent(t *testing.T) {
	parent := historyAliasedParent(t)
	source := filepath.Join(parent, "source")
	require.NoError(t, os.MkdirAll(filepath.Join(source, "bin"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(source, "bin", "cjc"), []byte("compiler"), 0o755))
	require.NoError(t, os.Symlink("cjc", filepath.Join(source, "bin", "compiler")))
	realSource, err := filepath.EvalSymlinks(source)
	require.NoError(t, err)
	want, err := historyTreeDigest(t.Context(), realSource)
	require.NoError(t, err)

	t.Run("digest", func(t *testing.T) {
		got, err := historyTreeDigest(t.Context(), source)
		require.NoError(t, err)
		require.Equal(t, want, got, "an ancestor alias must not change the content digest")
	})
	t.Run("copy", func(t *testing.T) {
		dest := filepath.Join(parent, "copy")
		require.NoError(t, copyHistoryTree(t.Context(), source, dest, nil))
		got, err := historyTreeDigest(t.Context(), dest)
		require.NoError(t, err)
		require.Equal(t, want, got)
		link, err := os.Readlink(filepath.Join(dest, "bin", "compiler"))
		require.NoError(t, err)
		require.Equal(t, "cjc", link, "copy must preserve the relative link")
		contents, err := os.ReadFile(filepath.Join(dest, "bin", "compiler"))
		require.NoError(t, err)
		require.Equal(t, "compiler", string(contents))
	})
}

func TestHistoryRejectsUnsafeLinksUnderSymlinkedParent(t *testing.T) {
	for _, kind := range []string{"outside", "absolute", "root", "dangling", "cycle"} {
		t.Run(kind, func(t *testing.T) {
			parent := historyAliasedParent(t)
			source := filepath.Join(parent, "source")
			outside := filepath.Join(parent, "source-sibling")
			require.NoError(t, os.Mkdir(source, 0o755))
			require.NoError(t, os.Mkdir(outside, 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(outside, "payload"), []byte("outside"), 0o644))
			var message string
			switch kind {
			case "outside":
				require.NoError(t, os.Symlink("../source-sibling/payload", filepath.Join(source, "link")))
				message = "history link leaves its content root"
			case "absolute":
				require.NoError(t, os.WriteFile(filepath.Join(source, "payload"), []byte("inside"), 0o644))
				require.NoError(t, os.Symlink(filepath.Join(source, "payload"), filepath.Join(source, "link")))
				message = "history link must be relative"
			case "root":
				link := filepath.Join(parent, "linked-root")
				require.NoError(t, os.Symlink("source", link))
				source = link
				message = "history content root must be"
			case "dangling":
				require.NoError(t, os.Symlink("missing", filepath.Join(source, "link")))
			case "cycle":
				require.NoError(t, os.Symlink("link", filepath.Join(source, "link")))
			}
			_, err := historyTreeDigest(t.Context(), source)
			require.Error(t, err)
			if message != "" {
				require.ErrorContains(t, err, message)
			}
			err = copyHistoryTree(t.Context(), source, filepath.Join(parent, "copy"), nil)
			require.Error(t, err)
			if message != "" {
				require.ErrorContains(t, err, message)
			}
		})
	}
}

func TestHistoryRollbackInternalLinksUnderSymlinkedParent(t *testing.T) {
	home := filepath.Join(historyAliasedParent(t), "home")
	configtest.Isolate(t, home)
	t.Setenv(config.EnvDistServer, "")
	require.NoError(t, config.EnsureDirs())
	selectRelease(t, "1.0.0")
	require.NoError(t, Install(t.Context(), InstallRequest{Toolchain: "sts"}, Options{}))
	sdk := filepath.Join(home, "toolchains", "sts")
	link := filepath.Join(sdk, "bin", "compiler")
	require.NoError(t, os.Symlink("cjc", link))
	want, err := os.ReadFile(link)
	require.NoError(t, err)
	selectRelease(t, "2.0.0")
	_, err = UpdateInstalled(t.Context(), parse(t, "sts"), Options{})
	require.NoError(t, err)
	entries, err := ListHistory(t.Context(), "sts")
	require.NoError(t, err)
	require.Len(t, entries, 1)
	_, err = RollbackHistory(t.Context(), "sts", entries[0].ID, Options{})
	require.NoError(t, err)
	require.Equal(t, "sts-1.0.0", readRelease(t, home, "sts").Release)
	target, err := os.Readlink(link)
	require.NoError(t, err)
	require.Equal(t, "cjc", target)
	got, err := os.ReadFile(link)
	require.NoError(t, err)
	require.Equal(t, want, got)
}
