package dist

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/toolchain"
	goversion "github.com/hashicorp/go-version"
)

type DownloadInfo struct {
	Name   string `json:"name"`
	SHA256 string `json:"sha256"`
	URL    string `json:"url"`
}

// ComponentInfo is a toolchain add-on archive (docs / stdx / stdx-docs).
// SHA256 is optional for upstream compatibility and available to controlled
// distribution sources.
type ComponentInfo struct {
	Name   string `json:"name"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256,omitempty"`
}

// ComponentSet holds the component download links for a single version: the
// main docs and stdx-docs archives (one each), plus the stdx binaries keyed by
// archive platform token (e.g. "linux-x64", "ohos-aarch64").
type ComponentSet struct {
	Docs     *ComponentInfo           `json:"docs,omitempty"`
	StdxDocs *ComponentInfo           `json:"stdx-docs,omitempty"`
	Stdx     map[string]ComponentInfo `json:"stdx,omitempty"`
}

type ChannelInfo struct {
	Latest     string                             `json:"latest"`
	Versions   map[string]map[string]DownloadInfo `json:"versions"`             // version -> platform -> info
	Components map[string]ComponentSet            `json:"components,omitempty"` // version -> component set
}

type Manifest struct {
	Channels struct {
		LTS     ChannelInfo  `json:"lts"`
		STS     ChannelInfo  `json:"sts"`
		Nightly *ChannelInfo `json:"nightly,omitempty"`
	} `json:"channels"`
}

// ErrManifestChannelMissing reports a channel absent from a parsed manifest.
var ErrManifestChannelMissing = errors.New("manifest channel is missing")

func ParseManifest(data []byte) (*Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}
	if err := m.validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

// ParseChannelManifest parses and validates one channel document.
func ParseChannelManifest(data []byte, channel toolchain.Channel) (*ChannelInfo, error) {
	var info ChannelInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, fmt.Errorf("failed to parse %s manifest: %w", channel, err)
	}
	if err := validateChannel(channel, info); err != nil {
		return nil, err
	}
	return &info, nil
}

func (m *Manifest) GetDownloadInfo(channel toolchain.Channel, version, tuple string) (*DownloadInfo, error) {
	ch, err := m.getChannel(channel)
	if err != nil {
		return nil, err
	}

	platforms, ok := ch.Versions[version]
	if !ok {
		return nil, &cjverr.VersionNotFoundError{Version: version}
	}

	info, ok := platforms[tuple]
	if !ok {
		return nil, &cjverr.VersionNotAvailableError{Version: version, Target: tuple}
	}

	return &info, nil
}

// ComponentDownload returns the download link for a component archive. comp is
// the component name ("stdx", "docs", "stdx-docs"); stdxPlatform is the stdx
// archive platform token (e.g. "linux-x64") and is consulted only for stdx.
// It returns ComponentNotPublishedError when the manifest carries no link for
// the requested version / component / platform.
func (m *Manifest) ComponentDownload(channel toolchain.Channel, version, comp, stdxPlatform string) (*ComponentInfo, error) {
	ch, err := m.getChannel(channel)
	if err != nil {
		return nil, err
	}
	set, ok := ch.Components[version]
	if !ok {
		return nil, &cjverr.ComponentNotPublishedError{Component: comp, Version: version}
	}
	switch comp {
	case "stdx":
		info, ok := set.Stdx[stdxPlatform]
		if !ok {
			return nil, &cjverr.ComponentNotPublishedError{Component: comp, Version: version, Target: stdxPlatform}
		}
		return &info, nil
	case "docs":
		if set.Docs == nil {
			return nil, &cjverr.ComponentNotPublishedError{Component: comp, Version: version}
		}
		return set.Docs, nil
	case "stdx-docs":
		if set.StdxDocs == nil {
			return nil, &cjverr.ComponentNotPublishedError{Component: comp, Version: version}
		}
		return set.StdxDocs, nil
	default:
		return nil, &cjverr.UnknownComponentError{Name: comp}
	}
}

// HasComponents reports whether the channel publishes any component for version.
func (m *Manifest) HasComponents(channel toolchain.Channel, version string) bool {
	ch, err := m.getChannel(channel)
	if err != nil {
		return false
	}
	_, ok := ch.Components[version]
	return ok
}

// HasChannel reports whether the manifest provides metadata for channel.
func (m *Manifest) HasChannel(channel toolchain.Channel) bool {
	if m == nil {
		return false
	}
	switch channel {
	case toolchain.LTS, toolchain.STS:
		return true
	case toolchain.Nightly:
		return m.Channels.Nightly != nil
	default:
		return false
	}
}

func (m *Manifest) GetLatestVersion(channel toolchain.Channel) (string, error) {
	ch, err := m.getChannel(channel)
	if err != nil {
		return "", err
	}
	if ch.Latest == "" {
		return "", fmt.Errorf("channel %s has no latest version", channel)
	}
	return ch.Latest, nil
}

// ListVersions returns the versions of the given channel, sorted descending by
// semver (entries that fail to parse fall back to lexical order). When
// tuple is non-empty, only versions that have a build for that key are
// returned; the empty string disables platform filtering.
func (m *Manifest) ListVersions(channel toolchain.Channel, tuple string) ([]string, error) {
	ch, err := m.getChannel(channel)
	if err != nil {
		return nil, err
	}
	out := make([]string, 0, len(ch.Versions))
	for v, platforms := range ch.Versions {
		if tuple != "" {
			if _, ok := platforms[tuple]; !ok {
				continue
			}
		}
		out = append(out, v)
	}
	sortVersionsDesc(out)
	return out, nil
}

// VersionsByTuple returns a map from each observed target tuple in the
// channel to the descending semver-sorted list of versions that ship a build
// for it. Used by the --all-platforms listing mode.
func (m *Manifest) VersionsByTuple(channel toolchain.Channel) (map[string][]string, error) {
	ch, err := m.getChannel(channel)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]string)
	for v, tuples := range ch.Versions {
		for tuple := range tuples {
			result[tuple] = append(result[tuple], v)
		}
	}
	for tuple := range result {
		sortVersionsDesc(result[tuple])
	}
	return result, nil
}

func sortVersionsDesc(versions []string) {
	sort.SliceStable(versions, func(i, j int) bool {
		return compareSemVerDesc(versions[i], versions[j])
	})
}

// compareSemVerDesc reports whether a should sort before b in a descending
// listing. Mirrors internal/toolchain/toolchain.go:compareSemVer semantics:
// values that parse as semver are compared as such; unparseable values fall
// back to string comparison; parseable values sort ahead of unparseable ones.
func compareSemVerDesc(a, b string) bool {
	va, aErr := goversion.NewVersion(a)
	vb, bErr := goversion.NewVersion(b)
	switch {
	case aErr == nil && bErr == nil:
		return va.Compare(vb) > 0
	case aErr == nil:
		return true
	case bErr == nil:
		return false
	default:
		return strings.Compare(a, b) > 0
	}
}

// FindVersionChannel searches for a version across channels (LTS first, then STS).
func (m *Manifest) FindVersionChannel(version string) (toolchain.Channel, error) {
	if _, ok := m.Channels.LTS.Versions[version]; ok {
		return toolchain.LTS, nil
	}
	if _, ok := m.Channels.STS.Versions[version]; ok {
		return toolchain.STS, nil
	}
	return toolchain.UnknownChannel, &cjverr.VersionNotFoundError{Version: version}
}

func (m *Manifest) getChannel(ch toolchain.Channel) (*ChannelInfo, error) {
	switch ch {
	case toolchain.LTS:
		return &m.Channels.LTS, nil
	case toolchain.STS:
		return &m.Channels.STS, nil
	case toolchain.Nightly:
		if m.Channels.Nightly == nil {
			return nil, fmt.Errorf("channel %s: %w", ch, ErrManifestChannelMissing)
		}
		return m.Channels.Nightly, nil
	default:
		return nil, &cjverr.UnknownChannelError{Channel: ch.String()}
	}
}

func (m *Manifest) validate() error {
	if err := validateChannel(toolchain.LTS, m.Channels.LTS); err != nil {
		return err
	}
	if err := validateChannel(toolchain.STS, m.Channels.STS); err != nil {
		return err
	}
	if m.Channels.Nightly != nil {
		if err := validateChannel(toolchain.Nightly, *m.Channels.Nightly); err != nil {
			return err
		}
	}
	return nil
}

func validateChannel(channel toolchain.Channel, ch ChannelInfo) error {
	label := channel.String()
	if ch.Latest == "" {
		return fmt.Errorf("channel %s has no latest version", label)
	}
	if len(ch.Versions) == 0 {
		return fmt.Errorf("channel %s has no versions", label)
	}
	if _, ok := ch.Versions[ch.Latest]; !ok {
		return fmt.Errorf("channel %s latest version %s is missing from versions", label, ch.Latest)
	}
	for version, platforms := range ch.Versions {
		if len(platforms) == 0 {
			return fmt.Errorf("channel %s version %s has no platforms", label, version)
		}
		for platform, info := range platforms {
			if channel == toolchain.Nightly && info.SHA256 == "" {
				if info.Name == "" {
					return fmt.Errorf("channel %s version %s platform %s has empty name", label, version, platform)
				}
				if info.URL == "" {
					return fmt.Errorf("channel %s version %s platform %s has empty url", label, version, platform)
				}
				continue
			}
			if err := validateDownloadInfo(label, version, platform, info); err != nil {
				return err
			}
		}
	}
	for version, set := range ch.Components {
		if set.Docs != nil {
			if err := validateComponentInfo(label, version, "docs", *set.Docs); err != nil {
				return err
			}
		}
		if set.StdxDocs != nil {
			if err := validateComponentInfo(label, version, "stdx-docs", *set.StdxDocs); err != nil {
				return err
			}
		}
		for platform, info := range set.Stdx {
			if err := validateComponentInfo(label, version, "stdx/"+platform, info); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateComponentInfo(channel, version, component string, info ComponentInfo) error {
	if info.URL == "" {
		return fmt.Errorf("channel %s version %s component %s has empty url", channel, version, component)
	}
	if info.SHA256 == "" {
		return nil
	}
	if len(info.SHA256) != 64 {
		return fmt.Errorf("channel %s version %s component %s has invalid sha256 length", channel, version, component)
	}
	if _, err := hex.DecodeString(info.SHA256); err != nil {
		return fmt.Errorf("channel %s version %s component %s has invalid sha256: %w", channel, version, component, err)
	}
	return nil
}

func validateDownloadInfo(channel, version, platform string, info DownloadInfo) error {
	if info.Name == "" {
		return fmt.Errorf("channel %s version %s platform %s has empty name", channel, version, platform)
	}
	if info.URL == "" {
		return fmt.Errorf("channel %s version %s platform %s has empty url", channel, version, platform)
	}
	if len(info.SHA256) != 64 {
		return fmt.Errorf("channel %s version %s platform %s has invalid sha256 length", channel, version, platform)
	}
	if _, err := hex.DecodeString(info.SHA256); err != nil {
		return fmt.Errorf("channel %s version %s platform %s has invalid sha256: %w", channel, version, platform, err)
	}
	return nil
}
