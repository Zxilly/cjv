package target

import (
	"fmt"
	"regexp"
	"runtime"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
)

var (
	targetRE = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

	hostByGo = map[string]TupleParts{
		"windows-amd64": {Host: "win32-x64", NightlyOS: "windows", NightlyArch: "x64"},
		"darwin-arm64":  {Host: "darwin-arm64", NightlyOS: "mac", NightlyArch: "aarch64"},
		"darwin-amd64":  {Host: "darwin-x64", NightlyOS: "mac", NightlyArch: "x64"},
		"linux-arm64":   {Host: "linux-arm64", NightlyOS: "linux", NightlyArch: "aarch64"},
		"linux-amd64":   {Host: "linux-x64", NightlyOS: "linux", NightlyArch: "x64"},
	}

	hostByTuple = map[string]TupleParts{
		"win32-x64":    {Host: "win32-x64", NightlyOS: "windows", NightlyArch: "x64"},
		"darwin-arm64": {Host: "darwin-arm64", NightlyOS: "mac", NightlyArch: "aarch64"},
		"darwin-x64":   {Host: "darwin-x64", NightlyOS: "mac", NightlyArch: "x64"},
		"linux-arm64":  {Host: "linux-arm64", NightlyOS: "linux", NightlyArch: "aarch64"},
		"linux-x64":    {Host: "linux-x64", NightlyOS: "linux", NightlyArch: "x64"},
		"ohos-arm64":   {Host: "ohos-arm64", NightlyOS: "ohos", NightlyArch: "aarch64"},
		"ohos-x64":     {Host: "ohos-x64", NightlyOS: "ohos", NightlyArch: "x64"},
	}

	hostTuples = []string{
		"win32-x64",
		"darwin-arm64",
		"darwin-x64",
		"linux-arm64",
		"linux-x64",
		"ohos-arm64",
		"ohos-x64",
	}
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
	parts, ok := hostByGo[goos+"-"+goarch]
	if !ok {
		return "", &cjverr.UnsupportedPlatformError{OS: goos, Arch: goarch}
	}
	return parts.Host, nil
}

// CurrentHostTuple returns the host target tuple for the current process,
// honoring the optional defaultHost override (in goos-goarch form).
func CurrentHostTuple(defaultHost string) (string, error) {
	if defaultHost != "" {
		parts := strings.SplitN(defaultHost, "-", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid default_host format: %q (expected goos-goarch)", defaultHost)
		}
		return HostTuple(parts[0], parts[1])
	}
	return HostTuple(runtime.GOOS, runtime.GOARCH)
}

// BuildTuple composes a target tuple from a bare host tuple and an optional
// environment (the trailing component, e.g. "ohos", "gnu", "musl"). An empty
// environment returns hostTuple unchanged.
func BuildTuple(hostTuple, environment string) (string, error) {
	parts, err := ParseTuple(hostTuple)
	if err != nil {
		return "", err
	}
	if parts.Environment != "" {
		return "", fmt.Errorf("host tuple %q must not include an environment suffix", hostTuple)
	}
	normalized, err := Normalize(environment)
	if err != nil {
		return "", err
	}
	if normalized == "" {
		return hostTuple, nil
	}
	return hostTuple + "-" + normalized, nil
}

// HostPartOf strips any target suffix from a target tuple, returning just the
// host portion (e.g. "linux-x64-ohos" → "linux-x64"). Useful for components
// like stdx that are not target-specific.
func HostPartOf(tuple string) (string, error) {
	parts, err := ParseTuple(tuple)
	if err != nil {
		return "", err
	}
	return parts.Host, nil
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
