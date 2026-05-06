package component

import (
	"fmt"
	"net/url"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/dist"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// DefaultStdxReleaseBaseURL is the gitcode releases prefix for the
// cangjie_stdx repository (used for LTS/STS stdx and stdx-docs archives).
const DefaultStdxReleaseBaseURL = "https://gitcode.com/Cangjie/cangjie_stdx/releases/download"

// DefaultDocsBundleBaseURL is the github releases prefix for the cangjie-docs
// bundle that publishes LTS / STS main documentation tarballs.
const DefaultDocsBundleBaseURL = "https://github.com/Zxilly/cangjie-docs-bundle/releases/download"

// Test overrides for the LTS/STS download URLs; empty falls back to the default.
var (
	stdxReleaseBaseOverride string
	docsBundleBaseOverride  string
)

func stdxReleaseBase() string {
	if stdxReleaseBaseOverride != "" {
		return stdxReleaseBaseOverride
	}
	return DefaultStdxReleaseBaseURL
}

func docsBundleBase() string {
	if docsBundleBaseOverride != "" {
		return docsBundleBaseOverride
	}
	return DefaultDocsBundleBaseURL
}

// ResolveAssetURL requires platformKey for stdx (host key, no target suffix)
// and ignores it for docs / stdx-docs. Returns ComponentNotAvailableForChannelError
// when the component does not ship offline for tc.Channel.
func ResolveAssetURL(spec Spec, tc toolchain.ToolchainName, platformKey string) (string, error) {
	if !spec.SupportsChannel(tc.Channel) {
		return "", &cjverr.ComponentNotAvailableForChannelError{
			Component: string(spec.Name),
			Channel:   tc.Channel.String(),
		}
	}
	if tc.Version == "" {
		return "", fmt.Errorf("component %q requires a resolved toolchain version", spec.Name)
	}

	switch spec.Name {
	case Stdx:
		return stdxURL(tc, platformKey)
	case Docs:
		return docsURL(tc)
	case StdxDocs:
		return stdxDocsURL(tc)
	default:
		return "", &cjverr.UnknownComponentError{Name: string(spec.Name)}
	}
}

func stdxURL(tc toolchain.ToolchainName, platformKey string) (string, error) {
	if platformKey == "" {
		return "", fmt.Errorf("stdx requires a host platform key")
	}
	hostKey, err := sdktarget.HostKeyOnly(platformKey)
	if err != nil {
		return "", err
	}
	if hostKey != platformKey {
		return "", fmt.Errorf("stdx is not target-specific; got toolchain key %q", platformKey)
	}
	parts, err := sdktarget.ParseToolchainKey(hostKey)
	if err != nil {
		return "", err
	}
	ext := dist.ArchiveExt(dist.NightlyGOOS(parts.NightlyOS))
	asset := fmt.Sprintf("cangjie-stdx-%s-%s-%s.1%s",
		parts.NightlyOS, parts.NightlyArch, tc.Version, ext)

	switch tc.Channel {
	case toolchain.Nightly:
		return joinReleaseURL(dist.DefaultNightlyBaseURL, tc.Version, asset)
	default:
		return joinReleaseURL(stdxReleaseBase(), "v"+tc.Version, asset)
	}
}

// docsURL: nightly from the cangjie nightly_build release; LTS / STS from
// the cangjie-docs-bundle GitHub release, tag = bare version (no `v` prefix).
func docsURL(tc toolchain.ToolchainName) (string, error) {
	asset := fmt.Sprintf("cangjie-docs-html-%s.tar.gz", tc.Version)
	switch tc.Channel {
	case toolchain.Nightly:
		return joinReleaseURL(dist.DefaultNightlyBaseURL, tc.Version, asset)
	default:
		return joinReleaseURL(docsBundleBase(), tc.Version, asset)
	}
}

// stdxDocsURL: nightly from nightly_build; LTS / STS from cangjie_stdx,
// tag = `v{ver}.1`, asset suffix `.1`.
func stdxDocsURL(tc toolchain.ToolchainName) (string, error) {
	asset := fmt.Sprintf("cangjie-stdx-docs-html-%s.1.tar.gz", tc.Version)
	switch tc.Channel {
	case toolchain.Nightly:
		return joinReleaseURL(dist.DefaultNightlyBaseURL, tc.Version, asset)
	default:
		return joinReleaseURL(stdxReleaseBase(), "v"+tc.Version+".1", asset)
	}
}

func joinReleaseURL(base, tag, asset string) (string, error) {
	u, err := url.Parse(base)
	if err != nil {
		return "", fmt.Errorf("invalid base URL %q: %w", base, err)
	}
	return u.JoinPath(tag, asset).String(), nil
}
