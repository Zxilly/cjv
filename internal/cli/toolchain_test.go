package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToolchainLinkCommandCreatesCustomLinkAndProxyLinks(t *testing.T) {
	home := t.TempDir()
	target := t.TempDir()
	config.IsolateForTest(t, home)

	cjcPath := filepath.Join(target, "bin", proxy.PlatformBinaryName("cjc"))
	require.NoError(t, os.MkdirAll(filepath.Dir(cjcPath), 0o755))
	require.NoError(t, os.WriteFile(cjcPath, []byte("stub"), 0o755))

	err := toolchainLinkCmd.RunE(toolchainLinkCmd, []string{"my-sdk", target})

	require.NoError(t, err)
	assert.FileExists(t, filepath.Join(home, "bin", proxy.CjvBinaryName()))
	assert.FileExists(t, filepath.Join(home, "bin", proxy.PlatformBinaryName("cjc")))
	_, statErr := os.Lstat(filepath.Join(home, "toolchains", "my-sdk"))
	assert.NoError(t, statErr)

	err = toolchainLinkCmd.RunE(toolchainLinkCmd, []string{"my-sdk", target})
	require.Error(t, err)
}

func TestToolchainLinkCommandRejectsInvalidInputs(t *testing.T) {
	home := t.TempDir()
	config.IsolateForTest(t, home)

	require.Error(t, toolchainLinkCmd.RunE(toolchainLinkCmd, []string{"lts", t.TempDir()}))
	require.Error(t, toolchainLinkCmd.RunE(toolchainLinkCmd, []string{"bad/path", t.TempDir()}))
	require.Error(t, toolchainLinkCmd.RunE(toolchainLinkCmd, []string{"my-sdk", filepath.Join(t.TempDir(), "missing")}))
}
