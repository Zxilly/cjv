package proxy

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToolPath(t *testing.T) {
	assert.Equal(t, filepath.FromSlash("bin/cjc"), ToolRelativePath("cjc"))
	assert.Equal(t, filepath.FromSlash("bin/cjc-frontend"), ToolRelativePath("cjc-frontend"))
	assert.Equal(t, filepath.FromSlash("tools/bin/cjpm"), ToolRelativePath("cjpm"))
	assert.Equal(t, filepath.FromSlash("tools/bin/cjfmt"), ToolRelativePath("cjfmt"))
	assert.Equal(t, filepath.FromSlash("tools/bin/cjprof"), ToolRelativePath("cjprof"))
	assert.Equal(t, filepath.FromSlash("tools/bin/LSPServer"), ToolRelativePath("LSPServer"))
	if runtime.GOOS == "windows" {
		assert.Equal(t, filepath.FromSlash("bin/cjc"), ToolRelativePath("CJC"))
		assert.Equal(t, filepath.FromSlash("tools/bin/LSPServer"), ToolRelativePath("lspserver"))
	}
}

func TestIsProxyTool(t *testing.T) {
	assert.True(t, IsProxyTool("cjc"))
	assert.True(t, IsProxyTool("cjpm"))
	assert.True(t, IsProxyTool("LSPMacroServer"))
	if runtime.GOOS == "windows" {
		assert.True(t, IsProxyTool("CJC"))
		assert.True(t, IsProxyTool("lspserver"))
		assert.True(t, IsProxyTool("LSPMACROSERVER"))
	}
	assert.False(t, IsProxyTool("cjv"))
	assert.False(t, IsProxyTool("unknown"))
}

func TestAllProxyTools(t *testing.T) {
	tools := AllProxyTools()
	assert.Contains(t, tools, "cjc")
	assert.Contains(t, tools, "cjpm")
	assert.Len(t, tools, 13) // 2 bin + 11 tools/bin
}
