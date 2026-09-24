// Package sdktools describes the SDK tool layout: which tools a toolchain
// ships and where each one lives inside the toolchain directory, how the cjv
// binary and the tools are named per platform, how the proxy links under
// CJV_HOME/bin are created, and how an installed tool binary is located for a
// toolchain directory and target tuple. It sits below lifecycle, env and
// selfupdate so that all of them read the same layout.
package sdktools

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
	"cjprof":          "tools/bin/cjprof",
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
