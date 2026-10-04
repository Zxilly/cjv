package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestToolchainFileAbsolutePathResolution(t *testing.T) {
	IsolateForTest(t, t.TempDir())
	project, sdk := t.TempDir(), t.TempDir()
	path := filepath.Join(project, ToolchainFileName)
	require.NoError(t, os.WriteFile(path, []byte(fmt.Sprintf("[toolchain]\npath = '%s'\ncomponents = ['invalid']\ntargets = ['invalid target']\n", sdk)), 0644))
	settings := DefaultSettings()
	selected, err := ResolveToolchainConfig(&settings, project)
	require.NoError(t, err)
	require.Equal(t, sdk, selected.Name)
	require.Equal(t, SourceToolchainFile, selected.Source)
	require.Empty(t, selected.Targets)
	require.Empty(t, selected.Components)
	for _, body := range []string{"[toolchain]\npath = 'relative/sdk'\n", fmt.Sprintf("[toolchain]\npath = '%s'\nchannel = 'sts'\n", sdk)} {
		require.NoError(t, os.WriteFile(path, []byte(body), 0644))
		_, err := ParseToolchainFile(path)
		require.Error(t, err)
	}
}
