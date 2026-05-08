package proxy

import (
	"path/filepath"
	"runtime"
	"slices"
	"strings"
)

var toolPathMap = map[string]string{
	"cjc":             "bin/cjc",
	"cjc-frontend":    "bin/cjc-frontend",
	"cjpm":            "tools/bin/cjpm",
	"cjfmt":           "tools/bin/cjfmt",
	"cjlint":          "tools/bin/cjlint",
	"cjdb":            "tools/bin/cjdb",
	"cjcov":           "tools/bin/cjcov",
	"cjtrace-recover": "tools/bin/cjtrace-recover",
	"chir-dis":        "tools/bin/chir-dis",
	"hle":             "tools/bin/hle",
	"LSPServer":       "tools/bin/LSPServer",
	"LSPMacroServer":  "tools/bin/LSPMacroServer",
}

var allTools = func() []string {
	tools := make([]string, 0, len(toolPathMap))
	for name := range toolPathMap {
		tools = append(tools, name)
	}
	slices.Sort(tools)
	return tools
}()

var toolPathLookup = func() map[string]string {
	lookup := make(map[string]string, len(toolPathMap))
	for name, relPath := range toolPathMap {
		lookup[canonicalToolName(name)] = relPath
	}
	return lookup
}()

func canonicalToolName(name string) string {
	if runtime.GOOS == "windows" {
		return strings.ToLower(name)
	}
	return name
}

func ToolRelativePath(name string) string {
	return filepath.FromSlash(toolPathLookup[canonicalToolName(name)])
}

func IsProxyTool(name string) bool {
	_, ok := toolPathLookup[canonicalToolName(name)]
	return ok
}

func AllProxyTools() []string {
	return allTools
}
