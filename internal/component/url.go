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

// ResolveAssetURL requires a tuple for stdx — a host tuple selects the host
// stdx, a target tuple (with environment suffix) selects the matching
// cross-compile target stdx — and ignores tuple for docs / stdx-docs. Returns
// ComponentNotAvailableForChannelError when the component does not ship offline
// for tc.Channel.
func ResolveAssetURL(spec Spec, tc toolchain.ToolchainName, tuple string) (string, error) {
	if !spec.SupportsChannel(tc.Channel) {
		return "", &cjverr.ComponentNotAvailableForChannelError{
			Component: string(spec.Name),
			Channel:   tc.Channel.String(),
		}
	}
	if tc.Version == "" {
		return "", fmt.Errorf("component %q requires a resolved toolchain version", spec.Name)
	}
	if spec.assetURL == nil {
		return "", &cjverr.UnknownComponentError{Name: string(spec.Name)}
	}
	return spec.assetURL(tc, tuple)
}

func stdxURL(tc toolchain.ToolchainName, tuple string) (string, error) {
	if tuple == "" {
		return "", fmt.Errorf("stdx requires a host tuple")
	}
	parts, err := sdktarget.ParseTuple(tuple)
	if err != nil {
		return "", err
	}
	var platform string
	if parts.Environment == "" {
		// Host stdx: platform token derives from the host OS/arch.
		platform = parts.NightlyOS + "-" + parts.NightlyArch
	} else {
		// Cross-target stdx: platform token derives from the target environment.
		platform, err = sdktarget.StdxPlatformForEnvironment(parts.Environment)
		if err != nil {
			return "", err
		}
	}
	stdxVersion := tc.Version + ".1"
	asset := fmt.Sprintf("cangjie-stdx-%s-%s.zip", platform, stdxVersion)

	switch tc.Channel {
	case toolchain.Nightly:
		return joinReleaseURL(dist.DefaultNightlyBaseURL, tc.Version, asset)
	default:
		return joinReleaseURL(stdxReleaseBase(), "v"+stdxVersion, asset)
	}
}

// docsURL: nightly from the cangjie nightly_build release; LTS / STS from
// the cangjie-docs-bundle GitHub release, tag = bare version (no `v` prefix).
func docsURL(tc toolchain.ToolchainName, _ string) (string, error) {
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
func stdxDocsURL(tc toolchain.ToolchainName, _ string) (string, error) {
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
