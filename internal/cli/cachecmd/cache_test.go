package cachecmd

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Zxilly/cjv/internal/cli/clitest"
	"github.com/Zxilly/cjv/internal/config/configtest"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/stretchr/testify/require"
)

func TestCacheListDoesNotCreateDirectories(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	output, err := clitest.Execute(t, clitest.Root(Register), "cache", "list", "--json")
	require.NoError(t, err)
	var result map[string]json.RawMessage
	require.NoError(t, json.Unmarshal([]byte(output), &result))
	require.JSONEq(t, `[]`, string(result["entries"]))
	require.JSONEq(t, `1`, string(result["schema_version"]))
	require.NoDirExists(t, filepath.Join(home, "cache"))
}

func TestCachePruneRejectsInvalidAgeBeforeWriting(t *testing.T) {
	i18n.Init("en")
	t.Cleanup(func() { i18n.Init("en") })
	home := t.TempDir()
	configtest.Isolate(t, home)
	_, err := clitest.Execute(t, clitest.Root(Register), "cache", "prune", "--max-age=-1h")
	require.ErrorContains(t, err, "max-age must not be negative")
	require.NoFileExists(t, filepath.Join(home, "cache.lock"))
}

func TestCacheListReportsDamageAndCleanRecovers(t *testing.T) {
	home := t.TempDir()
	configtest.Isolate(t, home)
	directory := filepath.Join(home, "cache")
	require.NoError(t, os.MkdirAll(directory, 0o700))
	key := strings.Repeat("a", 64)
	require.NoError(t, os.WriteFile(filepath.Join(directory, key+".json"), []byte("{"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(directory, key+".data"), []byte("incomplete data"), 0o600))
	output, err := clitest.Execute(t, clitest.Root(Register), "cache", "list", "--json")
	require.NoError(t, err)
	var listed result
	require.NoError(t, json.Unmarshal([]byte(output), &listed))
	require.Len(t, listed.Entries, 1)
	require.Equal(t, "invalid", listed.Entries[0].Kind)
	require.Contains(t, listed.Entries[0].Problem, "invalid cache metadata")
	output, err = clitest.Execute(t, clitest.Root(Register), "cache", "list")
	require.NoError(t, err)
	require.Contains(t, output, "invalid cache metadata")
	output, err = clitest.Execute(t, clitest.Root(Register), "cache", "clean", "--json")
	require.NoError(t, err)
	var cleaned result
	require.NoError(t, json.Unmarshal([]byte(output), &cleaned))
	require.NotNil(t, cleaned.Removed)
	require.Equal(t, 1, *cleaned.Removed)
	require.NoFileExists(t, filepath.Join(directory, key+".data"))
	require.NoFileExists(t, filepath.Join(directory, key+".json"))
}
