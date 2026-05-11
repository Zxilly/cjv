package component

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/utils"
)

// DocsRoot returns the absolute documentation root for the toolchain.
// Individual doc components (docs / stdx-docs) live under their own subdirs
// (`main/`, `stdx/`) so they never overwrite each other.
func DocsRoot(roots Roots) string { return roots.DocsDir }

// docsSubdirFor returns the documentation subdirectory owned by a component
// (e.g. "main" for Docs, "stdx" for StdxDocs). Empty string for non-doc
// components.
func docsSubdirFor(name Name) string {
	spec, err := SpecFor(name)
	if err != nil || spec.Location.Anchor != AnchorDocs {
		return ""
	}
	return spec.Location.Subdir
}

// topicCandidate is one possible relative path under the docs root, paired
// with the doc component that supplies it. Multiple candidates per topic
// cope with mdBook's varying entry-page names across releases.
type topicCandidate struct {
	component Name
	relPath   string // forward-slash, relative to docs root
}

// topicTable maps a user-supplied topic to ordered candidates. The first
// candidate that exists on disk wins.
var topicTable = map[string][]topicCandidate{
	"stdx": {
		{StdxDocs, "libs_stdx/libs_overview.html"},
		{StdxDocs, "libs_stdx/index.html"},
		{StdxDocs, "libs_stdx/actors/actors_package_overview.html"},
	},
	"std": {
		{Docs, "libs/std/core/core_package_overview.html"},
		{Docs, "libs/std/index.html"},
	},
	"dev-guide": {
		{Docs, "dev-guide/source_zh_cn/first_understanding/install.html"},
		{Docs, "dev-guide/source_zh_cn/index.html"},
	},
	"book": {
		{Docs, "dev-guide/source_zh_cn/first_understanding/install.html"},
		{Docs, "dev-guide/source_zh_cn/index.html"},
	},
	"tools": {
		{Docs, "tools/source_zh_cn/command_line_overview.html"},
		{Docs, "tools/source_zh_cn/index.html"},
	},
}

// ResolveDocPath maps a topic to an existing HTML file under the toolchain's
// docs root. An empty topic falls back to the docs entry index, preferring
// the main docs over stdx-only docs when both are installed. Returns the
// absolute file path on success.
func ResolveDocPath(roots Roots, topic string) (string, error) {
	docsRoot := DocsRoot(roots)
	tcName := filepath.Base(roots.TcDir)

	installed := map[Name]bool{
		Docs:     IsInstalled(roots.TcDir, Docs),
		StdxDocs: IsInstalled(roots.TcDir, StdxDocs),
	}
	if !installed[Docs] && !installed[StdxDocs] {
		return "", &cjverr.DocsNotInstalledError{Toolchain: tcName}
	}

	if topic == "" || strings.EqualFold(topic, "index") {
		// Prefer the main docs entry; fall back to stdx-docs.
		for _, name := range []Name{Docs, StdxDocs} {
			if !installed[name] {
				continue
			}
			candidate := filepath.Join(docsRoot, docsSubdirFor(name), "index.html")
			if fileExists(candidate) {
				return candidate, nil
			}
		}
		return "", &cjverr.DocsNotInstalledError{Toolchain: tcName}
	}

	if cands, ok := topicTable[strings.ToLower(topic)]; ok {
		var preferred Name
		for _, c := range cands {
			if !installed[c.component] {
				if preferred == "" {
					preferred = c.component
				}
				continue
			}
			candidate := filepath.Join(docsRoot, docsSubdirFor(c.component), filepath.FromSlash(c.relPath))
			if fileExists(candidate) {
				return candidate, nil
			}
		}
		return "", &cjverr.DocsTopicNotFoundError{
			Toolchain:        tcName,
			Topic:            topic,
			MissingComponent: string(preferred),
		}
	}

	// Fall back to treating the topic as a relative path; search every
	// installed doc component's subtree.
	rel, ok := safeDocRel(topic)
	if !ok {
		return "", &cjverr.DocsTopicNotFoundError{
			Toolchain:        tcName,
			Topic:            topic,
			MissingComponent: "",
		}
	}
	for _, name := range []Name{Docs, StdxDocs} {
		if !installed[name] {
			continue
		}
		base := filepath.Join(docsRoot, docsSubdirFor(name))
		for _, candidate := range []string{
			filepath.Join(base, rel),
			filepath.Join(base, rel+".html"),
			filepath.Join(base, rel, "index.html"),
		} {
			if utils.IsPathUnder(base, candidate) && fileExists(candidate) {
				return candidate, nil
			}
		}
	}
	return "", &cjverr.DocsTopicNotFoundError{
		Toolchain:        tcName,
		Topic:            topic,
		MissingComponent: "",
	}
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func safeDocRel(topic string) (string, bool) {
	if strings.TrimSpace(topic) == "" {
		return "", false
	}
	rel := filepath.FromSlash(topic)
	if filepath.VolumeName(rel) != "" || filepath.IsAbs(rel) {
		return "", false
	}
	clean := filepath.Clean(rel)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", false
	}
	return clean, true
}
