package target

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
)

var (
	targetRE            = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	stdxArchByEnvSuffix = map[string]string{"aarch64": "aarch64", "arm64": "aarch64", "x64": "x64", "arm32": "arm32"}
)

// TupleParts decomposes a target tuple into its host portion plus an optional
// cross-compile target suffix. NightlyOS/NightlyArch carry the GitCode-side
// naming for the host portion (used to construct nightly download filenames).
type TupleParts struct {
	Host        string
	Environment string
	NightlyOS   string
	NightlyArch string
}

// Identity is a parsed SDK target tuple. It gives callers a structured view of
// the host tuple, optional cross-target environment, nightly archive naming, and
// stdx platform token without making each caller split or suffix-check strings.
type Identity struct {
	tuple string
	parts TupleParts
}

// ParseIdentity parses a full target tuple into a structured identity.
func ParseIdentity(tuple string) (Identity, error) {
	parts, err := ParseTuple(tuple)
	if err != nil {
		return Identity{}, err
	}
	return Identity{tuple: tuple, parts: parts}, nil
}

// Tuple returns the canonical tuple string used as manifest index key.
func (id Identity) Tuple() string {
	return id.tuple
}

// HostTuple returns the host portion of the tuple.
func (id Identity) HostTuple() string {
	return id.parts.Host
}

// Environment returns the optional cross-target environment suffix without its
// leading dash, e.g. "ohos-arm32".
func (id Identity) Environment() string {
	return id.parts.Environment
}

// EnvironmentSuffix returns the target environment with its leading dash, or an
// empty string for host SDKs. It is useful for archive stems.
func (id Identity) EnvironmentSuffix() string {
	if id.parts.Environment == "" {
		return ""
	}
	return "-" + id.parts.Environment
}

// IsTargetVariant reports whether this identity points at a cross-target SDK.
func (id Identity) IsTargetVariant() bool {
	return id.parts.Environment != ""
}

// NightlyOS returns the OS token used by upstream nightly SDK assets.
func (id Identity) NightlyOS() string {
	return id.parts.NightlyOS
}

// NightlyArch returns the architecture token used by upstream nightly SDK
// assets.
func (id Identity) NightlyArch() string {
	return id.parts.NightlyArch
}

// NightlyGOOS returns the Go GOOS equivalent for the nightly OS token.
func (id Identity) NightlyGOOS() string {
	return nightlyGOOS(id.parts.NightlyOS)
}

// NightlyFilename returns the upstream nightly SDK archive filename for this
// target identity and version.
func (id Identity) NightlyFilename(version string) string {
	return "cangjie-sdk-" + id.NightlyOS() + "-" + id.NightlyArch() + id.EnvironmentSuffix() + "-" + version + archiveExt(id.NightlyGOOS())
}

// StdxPlatform returns the cangjie_stdx archive platform token for this target.
// Host SDKs use the host nightly OS/arch token; target variants use the target
// environment suffix (with an aarch64 default when no arch suffix is present).
func (id Identity) StdxPlatform() (string, error) {
	if id.parts.Environment == "" {
		return id.parts.NightlyOS + "-" + id.parts.NightlyArch, nil
	}
	return StdxPlatformForEnvironment(id.parts.Environment)
}

// WithEnvironment returns a target-variant identity for environment. The
// receiver must be a host identity; pass an empty environment to keep it as-is.
func (id Identity) WithEnvironment(environment string) (Identity, error) {
	if id.IsTargetVariant() {
		return Identity{}, fmt.Errorf("host tuple %q must not include an environment suffix", id.tuple)
	}
	normalized, err := Normalize(environment)
	if err != nil {
		return Identity{}, err
	}
	if normalized == "" {
		return id, nil
	}
	return ParseIdentity(id.parts.Host + "-" + normalized)
}

// Normalize canonicalises a single environment value (the trailing component
// of a target tuple, e.g. "ohos", "gnu", "musl") to the manifest's expected
// lowercase-with-dashes form. An empty input returns "" with no error.
func Normalize(input string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(input))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	if normalized == "" {
		return "", nil
	}
	if !targetRE.MatchString(normalized) {
		return "", fmt.Errorf("invalid environment %q: must match /^[a-z0-9]+(?:-[a-z0-9]+)*$/", input)
	}
	if _, err := ParseTuple(normalized); err == nil {
		return "", fmt.Errorf("invalid environment %q: provide an environment such as 'ohos', not a full target tuple", input)
	}
	return normalized, nil
}

func NormalizeList(values []string) ([]string, error) {
	var targets []string
	seen := make(map[string]bool)
	for _, value := range values {
		for part := range strings.SplitSeq(value, ",") {
			if strings.TrimSpace(part) == "" {
				return nil, fmt.Errorf("target list contains an empty target")
			}
			normalized, err := Normalize(part)
			if err != nil {
				return nil, err
			}
			if normalized == "" {
				return nil, fmt.Errorf("target list contains an empty target")
			}
			if !seen[normalized] {
				seen[normalized] = true
				targets = append(targets, normalized)
			}
		}
	}
	return targets, nil
}

// HostTuple returns the canonical host target tuple for a (goos, goarch) pair,
// e.g. ("linux", "amd64") → "linux-x64".
func HostTuple(goos, goarch string) (string, error) {
	id, err := HostIdentity(goos, goarch)
	if err != nil {
		return "", err
	}
	return id.Tuple(), nil
}

// HostIdentity returns the structured target identity for a (goos, goarch)
// host pair.
func HostIdentity(goos, goarch string) (Identity, error) {
	parts, ok := hostByGo[goos+"-"+goarch]
	if !ok {
		return Identity{}, &cjverr.UnsupportedPlatformError{OS: goos, Arch: goarch}
	}
	return Identity{tuple: parts.Host, parts: parts}, nil
}

// CurrentHostTuple returns the host target tuple for the current process,
// honoring the optional defaultHost override (in goos-goarch form).
func CurrentHostTuple(defaultHost string) (string, error) {
	id, err := CurrentHostIdentity(defaultHost)
	if err != nil {
		return "", err
	}
	return id.Tuple(), nil
}

// CurrentHostIdentity returns the structured host target identity for the
// current process, honoring the optional defaultHost override (in goos-goarch
// form).
func CurrentHostIdentity(defaultHost string) (Identity, error) {
	if defaultHost != "" {
		parts := strings.SplitN(defaultHost, "-", 2)
		if len(parts) != 2 {
			return Identity{}, fmt.Errorf("invalid default_host format: %q (expected goos-goarch)", defaultHost)
		}
		return HostIdentity(parts[0], parts[1])
	}
	return HostIdentity(runtime.GOOS, runtime.GOARCH)
}

// CurrentTargetIdentity resolves the current/default host and applies an
// optional cross-target environment to it.
func CurrentTargetIdentity(defaultHost, environment string) (Identity, error) {
	host, err := CurrentHostIdentity(defaultHost)
	if err != nil {
		return Identity{}, err
	}
	return host.WithEnvironment(environment)
}

// BuildTuple composes a target tuple from a bare host tuple and an optional
// environment (the trailing component, e.g. "ohos", "gnu", "musl"). An empty
// environment returns hostTuple unchanged.
func BuildTuple(hostTuple, environment string) (string, error) {
	id, err := ParseIdentity(hostTuple)
	if err != nil {
		return "", err
	}
	withEnvironment, err := id.WithEnvironment(environment)
	if err != nil {
		return "", err
	}
	return withEnvironment.Tuple(), nil
}

// HostPartOf strips any target suffix from a target tuple, returning just the
// host portion (e.g. "linux-x64-ohos" → "linux-x64"). Useful for components
// like stdx that are not target-specific.
func HostPartOf(tuple string) (string, error) {
	id, err := ParseIdentity(tuple)
	if err != nil {
		return "", err
	}
	return id.HostTuple(), nil
}

// ParseTuple splits a target tuple into its host portion and optional target
// suffix. The result also carries nightly-side OS/arch metadata.
func ParseTuple(tuple string) (TupleParts, error) {
	for _, host := range hostTuples {
		if tuple == host || strings.HasPrefix(tuple, host+"-") {
			parts, ok := hostByHostTuple(host)
			if !ok {
				break
			}
			parts.Environment = strings.TrimPrefix(tuple, host)
			parts.Environment = strings.TrimPrefix(parts.Environment, "-")
			if tuple != host {
				if parts.Environment == "" || !targetRE.MatchString(parts.Environment) {
					return TupleParts{}, fmt.Errorf("invalid target suffix in target tuple: %s", tuple)
				}
			}
			return parts, nil
		}
	}
	return TupleParts{}, fmt.Errorf("unsupported target tuple: %s", tuple)
}

// SplitVariantSuffix peels a trailing target tuple off a version-like string
// (e.g. "1.0.5-linux-x64-ohos" → ("1.0.5", "linux-x64-ohos")). It returns the
// original input and an empty tuple when no recognised suffix is present.
func SplitVariantSuffix(versionWithMaybeTuple string) (version, tuple string) {
	for _, host := range hostTuples {
		marker := "-" + host
		if i := strings.LastIndex(versionWithMaybeTuple, marker); i >= 0 {
			candidate := versionWithMaybeTuple[i+1:]
			if _, err := ParseTuple(candidate); err == nil {
				return versionWithMaybeTuple[:i], candidate
			}
		}
	}
	return versionWithMaybeTuple, ""
}

func hostByHostTuple(host string) (TupleParts, bool) {
	parts, ok := hostByTuple[host]
	return parts, ok
}

// StdxPlatformForEnvironment maps a cross-compile target environment (the
// trailing suffix of a target tuple, e.g. "ohos", "ohos-x64", "ios-simulator-x64")
// to the stdx archive platform token "{os}-{arch}". Environments with no
// recognised trailing arch default to aarch64.
func StdxPlatformForEnvironment(environment string) (string, error) {
	env := strings.ToLower(strings.TrimSpace(environment))
	if env == "" {
		return "", fmt.Errorf("stdx target environment cannot be empty")
	}
	osPart, arch := env, "aarch64"
	if i := strings.LastIndex(env, "-"); i >= 0 {
		if a, ok := stdxArchByEnvSuffix[env[i+1:]]; ok {
			osPart, arch = env[:i], a
		}
	}
	if osPart == "" {
		return "", fmt.Errorf("invalid stdx target environment %q", environment)
	}
	return osPart + "-" + arch, nil
}

// StdxPlatformForTuple maps a full SDK target tuple to the stdx archive
// platform token.
func StdxPlatformForTuple(tuple string) (string, error) {
	id, err := ParseIdentity(tuple)
	if err != nil {
		return "", err
	}
	return id.StdxPlatform()
}

func nightlyGOOS(nightlyOS string) string {
	if nightlyOS == "mac" {
		return "darwin"
	}
	return nightlyOS
}

func archiveExt(goos string) string {
	if goos == "windows" {
		return ".zip"
	}
	return ".tar.gz"
}
