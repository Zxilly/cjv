package proxy

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestToolPath(t *testing.T) {
	assert.Equal(t, filepath.FromSlash("bin/cjc"), ToolRelativePath("cjc"))
	assert.Equal(t, filepath.FromSlash("bin/cjc-frontend"), ToolRelativePath("cjc-frontend"))
	assert.Equal(t, filepath.FromSlash("tools/bin/cjpm"), ToolRelativePath("cjpm"))
	assert.Equal(t, filepath.FromSlash("tools/bin/cjfmt"), ToolRelativePath("cjfmt"))
	assert.Equal(t, filepath.FromSlash("tools/bin/LSPServer"), ToolRelativePath("LSPServer"))
}

func TestIsProxyTool(t *testing.T) {
	assert.True(t, IsProxyTool("cjc"))
	assert.True(t, IsProxyTool("cjpm"))
	assert.True(t, IsProxyTool("LSPMacroServer"))
	assert.False(t, IsProxyTool("cjv"))
	assert.False(t, IsProxyTool("unknown"))
}

func TestAllProxyTools(t *testing.T) {
	tools := AllProxyTools()
	assert.Contains(t, tools, "cjc")
	assert.Contains(t, tools, "cjpm")
	assert.Len(t, tools, 12) // 2 bin + 10 tools/bin
}
