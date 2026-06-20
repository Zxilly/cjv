// Package component models toolchain add-ons (stdx, docs, stdx-docs). Each
// component is a separately downloaded archive whose extracted files are
// tracked through a per-component manifest so it can be uninstalled
// independently.
package component

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/toolchain"
)

type Name string

const (
	Stdx     Name = "stdx"
	Docs     Name = "docs"
	StdxDocs Name = "stdx-docs"
)

// InstallLocation keeps the layout abstract so binaries can land inside
// the toolchain tree while pure data like docs stays under <CJV_HOME>/docs/<tc>/.
type InstallLocation struct {
	Anchor InstallAnchor
	Subdir string
}

type InstallAnchor int

const (
	AnchorDocs InstallAnchor = iota + 1
	AnchorStdx
)

type Spec struct {
	Name              Name
	Location          InstallLocation
	StripTopLevel     bool
	SupportedChannels []toolchain.Channel
	// EnvVars maps an environment variable name to a subdirectory, relative to
	// the component's install root, that the variable should point at when the
	// component is installed. Empty for components that contribute no runtime
	// environment (docs, stdx-docs).
	EnvVars map[string]string
	// Linkable reports whether `cjv component link` can point this component at
	// a local directory. LinkChildren are the subdirectories that get linked
	// and that must exist in the source.
	Linkable     bool
	LinkChildren []string
}

func (s Spec) SupportsChannel(ch toolchain.Channel) bool {
	return slices.Contains(s.SupportedChannels, ch)
}

func (s Spec) String() string { return string(s.Name) }

var specs = map[Name]Spec{
	Stdx: {
		Name:              Stdx,
		Location:          InstallLocation{Anchor: AnchorStdx, Subdir: ""},
		StripTopLevel:     true,
		SupportedChannels: []toolchain.Channel{toolchain.LTS, toolchain.STS, toolchain.Nightly},
		EnvVars: map[string]string{
			EnvStdxDynamic: "dynamic",
			EnvStdxStatic:  "static",
		},
		Linkable:     true,
		LinkChildren: []string{"dynamic", "static"},
	},
	Docs: {
		Name:              Docs,
		Location:          InstallLocation{Anchor: AnchorDocs, Subdir: "main"},
		StripTopLevel:     false,
		SupportedChannels: []toolchain.Channel{toolchain.LTS, toolchain.STS, toolchain.Nightly},
	},
	StdxDocs: {
		Name:              StdxDocs,
		Location:          InstallLocation{Anchor: AnchorDocs, Subdir: "stdx"},
		StripTopLevel:     false,
		SupportedChannels: []toolchain.Channel{toolchain.LTS, toolchain.STS, toolchain.Nightly},
	},
}

// Roots bundles the install roots a component may write to:
//   - TcDir   = <toolchains>/<tc>/
//   - DocsDir = <CJV_HOME>/docs/<tc>/
//   - StdxDir = <CJV_HOME>/stdx/<tc>/
type Roots struct {
	TcDir   string
	DocsDir string
	StdxDir string
}

func RootsFor(tcName string) (Roots, error) {
	tcDirRoot, err := config.ToolchainsDir()
	if err != nil {
		return Roots{}, err
	}
	docsDir, err := config.DocsDirFor(tcName)
	if err != nil {
		return Roots{}, err
	}
	stdxDir, err := config.StdxDirFor(tcName)
	if err != nil {
		return Roots{}, err
	}
	return Roots{
		TcDir:   filepath.Join(tcDirRoot, tcName),
		DocsDir: docsDir,
		StdxDir: stdxDir,
	}, nil
}

func (s Spec) InstallRoot(roots Roots) string {
	var base string
	switch s.Location.Anchor {
	case AnchorDocs:
		base = roots.DocsDir
	case AnchorStdx:
		base = roots.StdxDir
	default:
		panic(fmt.Sprintf("component %q has unhandled anchor %d", s.Name, s.Location.Anchor))
	}
	if s.Location.Subdir == "" {
		return base
	}
	return filepath.Join(base, filepath.FromSlash(s.Location.Subdir))
}

func SpecFor(name Name) (Spec, error) {
	s, ok := specs[name]
	if !ok {
		return Spec{}, &cjverr.UnknownComponentError{Name: string(name)}
	}
	return s, nil
}

func KnownComponents() []Name {
	names := make([]Name, 0, len(specs))
	for n := range specs {
		names = append(names, n)
	}
	slices.Sort(names)
	return names
}

// AvailableComponents lists the components installable for tc on the given
// tuple. Availability is decided by channel support (and, for stdx, whether the
// tuple resolves to an stdx archive platform); it does not consult the manifest,
// so a component shown here may still report ComponentNotPublishedError at
// install time if the resolved version happens not to publish it.
func AvailableComponents(tc toolchain.ToolchainName, tuple string) []Name {
	if tc.Version == "" {
		return nil
	}
	var out []Name
	for _, n := range KnownComponents() {
		spec, err := SpecFor(n)
		if err != nil || !spec.SupportsChannel(tc.Channel) {
			continue
		}
		if n == Stdx {
			if _, err := stdxPlatform(tuple); err != nil {
				continue
			}
		}
		out = append(out, n)
	}
	return out
}

func ParseName(s string) (Name, error) {
	trimmed := strings.TrimSpace(s)
	if trimmed == "" {
		return "", &cjverr.UnknownComponentError{Name: s}
	}
	n := Name(strings.ToLower(trimmed))
	if _, ok := specs[n]; !ok {
		return "", &cjverr.UnknownComponentError{Name: s}
	}
	return n, nil
}

// NormalizeList splits comma-separated entries (matching `-c a,b -c c`),
// trims whitespace, and de-duplicates.
func NormalizeList(values []string) ([]Name, error) {
	var out []Name
	seen := make(map[Name]bool)
	for _, raw := range values {
		for part := range strings.SplitSeq(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			n, err := ParseName(part)
			if err != nil {
				return nil, err
			}
			if !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	return out, nil
}
