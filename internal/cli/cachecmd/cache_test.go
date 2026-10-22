package cachecmd

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/cli/clitest"
	"github.com/Zxilly/cjv/internal/config/configtest"
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
	home := t.TempDir()
	configtest.Isolate(t, home)
	_, err := clitest.Execute(t, clitest.Root(Register), "cache", "prune", "--max-age=-1h")
	require.ErrorContains(t, err, "max-age must not be negative")
	require.NoFileExists(t, filepath.Join(home, "cache.lock"))
}
