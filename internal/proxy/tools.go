package proxy

import (
	"path/filepath"
	"slices"
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

func ToolRelativePath(name string) string {
	return filepath.FromSlash(toolPathMap[name])
}

func IsProxyTool(name string) bool {
	_, ok := toolPathMap[name]
	return ok
}

func AllProxyTools() []string {
	return allTools
}
