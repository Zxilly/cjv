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

	hostByGo = map[string]ToolchainKeyParts{
		"windows-amd64": {HostKey: "win32-x64", NightlyOS: "windows", NightlyArch: "x64"},
		"darwin-arm64":  {HostKey: "darwin-arm64", NightlyOS: "mac", NightlyArch: "aarch64"},
		"darwin-amd64":  {HostKey: "darwin-x64", NightlyOS: "mac", NightlyArch: "x64"},
		"linux-arm64":   {HostKey: "linux-arm64", NightlyOS: "linux", NightlyArch: "aarch64"},
		"linux-amd64":   {HostKey: "linux-x64", NightlyOS: "linux", NightlyArch: "x64"},
	}

	hostByKey = map[string]ToolchainKeyParts{
		"win32-x64":    {HostKey: "win32-x64", NightlyOS: "windows", NightlyArch: "x64"},
		"darwin-arm64": {HostKey: "darwin-arm64", NightlyOS: "mac", NightlyArch: "aarch64"},
		"darwin-x64":   {HostKey: "darwin-x64", NightlyOS: "mac", NightlyArch: "x64"},
		"linux-arm64":  {HostKey: "linux-arm64", NightlyOS: "linux", NightlyArch: "aarch64"},
		"linux-x64":    {HostKey: "linux-x64", NightlyOS: "linux", NightlyArch: "x64"},
		"ohos-arm64":   {HostKey: "ohos-arm64", NightlyOS: "ohos", NightlyArch: "aarch64"},
		"ohos-x64":     {HostKey: "ohos-x64", NightlyOS: "ohos", NightlyArch: "x64"},
	}

	hostKeys = []string{
		"win32-x64",
		"darwin-arm64",
		"darwin-x64",
		"linux-arm64",
		"linux-x64",
		"ohos-arm64",
		"ohos-x64",
	}
)

type ToolchainKeyParts struct {
	HostKey     string
	Target      string
	NightlyOS   string
	NightlyArch string
}

func Normalize(input string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(input))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	if normalized == "" {
		return "", nil
	}
	if !targetRE.MatchString(normalized) {
		return "", fmt.Errorf("invalid target %q: target must match /^[a-z0-9]+(?:-[a-z0-9]+)*$/", input)
	}
	if _, err := ParseToolchainKey(normalized); err == nil {
		return "", fmt.Errorf("invalid target %q: provide a target suffix such as 'ohos', not a full toolchain key", input)
	}
	return normalized, nil
}

func NormalizeList(values []string) ([]string, error) {
	var targets []string
	seen := make(map[string]bool)
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
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

func HostKey(goos, goarch string) (string, error) {
	parts, ok := hostByGo[goos+"-"+goarch]
	if !ok {
		return "", &cjverr.UnsupportedPlatformError{OS: goos, Arch: goarch}
	}
	return parts.HostKey, nil
}

func CurrentHostKey(defaultHost string) (string, error) {
	if defaultHost != "" {
		parts := strings.SplitN(defaultHost, "-", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid default_host format: %q (expected goos-goarch)", defaultHost)
		}
		return HostKey(parts[0], parts[1])
	}
	return HostKey(runtime.GOOS, runtime.GOARCH)
}

func ToolchainKey(hostKey, target string) (string, error) {
	parts, err := ParseToolchainKey(hostKey)
	if err != nil {
		return "", err
	}
	if parts.Target != "" {
		return "", fmt.Errorf("host key %q must not include a target suffix", hostKey)
	}
	normalized, err := Normalize(target)
	if err != nil {
		return "", err
	}
	if normalized == "" {
		return hostKey, nil
	}
	return hostKey + "-" + normalized, nil
}

func ParseToolchainKey(key string) (ToolchainKeyParts, error) {
	for _, hostKey := range hostKeys {
		if key == hostKey || strings.HasPrefix(key, hostKey+"-") {
			parts, ok := hostByHostKey(hostKey)
			if !ok {
				break
			}
			parts.Target = strings.TrimPrefix(key, hostKey)
			parts.Target = strings.TrimPrefix(parts.Target, "-")
			if key != hostKey {
				if parts.Target == "" || !targetRE.MatchString(parts.Target) {
					return ToolchainKeyParts{}, fmt.Errorf("invalid target suffix in toolchain key: %s", key)
				}
			}
			return parts, nil
		}
	}
	return ToolchainKeyParts{}, fmt.Errorf("unsupported toolchain key: %s", key)
}

func SplitVariantSuffix(versionWithMaybePlatform string) (version, platformKey string) {
	for _, hostKey := range hostKeys {
		marker := "-" + hostKey
		if i := strings.LastIndex(versionWithMaybePlatform, marker); i >= 0 {
			candidate := versionWithMaybePlatform[i+1:]
			if _, err := ParseToolchainKey(candidate); err == nil {
				return versionWithMaybePlatform[:i], candidate
			}
		}
	}
	return versionWithMaybePlatform, ""
}

func hostByHostKey(hostKey string) (ToolchainKeyParts, bool) {
	parts, ok := hostByKey[hostKey]
	return parts, ok
}
