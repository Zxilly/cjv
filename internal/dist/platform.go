package dist

import sdktarget "github.com/Zxilly/cjv/internal/target"

type platformMapping struct {
	JSONKey     string // platform key used in sdk-versions.json
	NightlyOS   string // OS segment in nightly filenames
	NightlyArch string // arch segment in nightly filenames
}

func lookup(goos, goarch string) (platformMapping, error) {
	hostKey, err := sdktarget.HostKey(goos, goarch)
	if err != nil {
		return platformMapping{}, err
	}
	return lookupPlatformKey(hostKey)
}

func lookupPlatformKey(platformKey string) (platformMapping, error) {
	parts, err := sdktarget.ParseToolchainKey(platformKey)
	if err != nil {
		return platformMapping{}, err
	}
	return platformMapping{
		JSONKey:     platformKey,
		NightlyOS:   parts.NightlyOS,
		NightlyArch: parts.NightlyArch,
	}, nil
}

func PlatformKey(goos, goarch string) (string, error) {
	m, err := lookup(goos, goarch)
	if err != nil {
		return "", err
	}
	return m.JSONKey, nil
}

// CurrentPlatformKey returns the platform key. If defaultHost is non-empty,
// it is used directly (format: "goos-goarch"); otherwise runtime values are used.
func CurrentPlatformKey(defaultHost string) (string, error) {
	return sdktarget.CurrentHostKey(defaultHost)
}

func CurrentPlatformKeyWithTarget(defaultHost, target string) (string, error) {
	hostKey, err := CurrentPlatformKey(defaultHost)
	if err != nil {
		return "", err
	}
	return sdktarget.ToolchainKey(hostKey, target)
}

func NightlyFilename(goos, goarch, version string) (string, error) {
	m, err := lookup(goos, goarch)
	if err != nil {
		return "", err
	}
	ext := ArchiveExt(goos)
	return "cangjie-sdk-" + m.NightlyOS + "-" + m.NightlyArch + "-" + version + ext, nil
}

func NightlyFilenameForPlatform(platformKey, version string) (string, error) {
	m, err := lookupPlatformKey(platformKey)
	if err != nil {
		return "", err
	}
	parts, err := sdktarget.ParseToolchainKey(platformKey)
	if err != nil {
		return "", err
	}
	targetPart := ""
	if parts.Target != "" {
		targetPart = "-" + parts.Target
	}
	ext := ArchiveExt(nightlyGOOS(m.NightlyOS))
	return "cangjie-sdk-" + m.NightlyOS + "-" + m.NightlyArch + targetPart + "-" + version + ext, nil
}

func nightlyGOOS(nightlyOS string) string {
	if nightlyOS == "windows" {
		return "windows"
	}
	if nightlyOS == "mac" {
		return "darwin"
	}
	return nightlyOS
}

func ArchiveExt(goos string) string {
	if goos == "windows" {
		return ".zip"
	}
	return ".tar.gz"
}
