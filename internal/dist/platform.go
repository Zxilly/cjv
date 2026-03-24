package dist

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
)

type platformMapping struct {
	JSONKey      string // platform key used in sdk-versions.json
	NightlyOS   string // OS segment in nightly filenames
	NightlyArch string // arch segment in nightly filenames
}

var platformMap = map[string]platformMapping{
	"windows-amd64": {JSONKey: "win32-x64", NightlyOS: "windows", NightlyArch: "x64"},
	"darwin-arm64":  {JSONKey: "darwin-arm64", NightlyOS: "mac", NightlyArch: "aarch64"},
	"darwin-amd64":  {JSONKey: "darwin-x64", NightlyOS: "mac", NightlyArch: "x64"}, // not all channels provide darwin-x64 builds
	"linux-amd64":   {JSONKey: "linux-x64", NightlyOS: "linux", NightlyArch: "x64"},
	"linux-arm64":   {JSONKey: "linux-arm64", NightlyOS: "linux", NightlyArch: "aarch64"},
}

func lookup(goos, goarch string) (platformMapping, error) {
	key := goos + "-" + goarch
	m, ok := platformMap[key]
	if !ok {
		return platformMapping{}, &cjverr.UnsupportedPlatformError{OS: goos, Arch: goarch}
	}
	return m, nil
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
	if defaultHost != "" {
		parts := strings.SplitN(defaultHost, "-", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid default_host format: %q (expected goos-goarch)", defaultHost)
		}
		return PlatformKey(parts[0], parts[1])
	}
	return PlatformKey(runtime.GOOS, runtime.GOARCH)
}

func NightlyFilename(goos, goarch, version string) (string, error) {
	m, err := lookup(goos, goarch)
	if err != nil {
		return "", err
	}
	ext := ArchiveExt(goos)
	return "cangjie-sdk-" + m.NightlyOS + "-" + m.NightlyArch + "-" + version + ext, nil
}

func ArchiveExt(goos string) string {
	if goos == "windows" {
		return ".zip"
	}
	return ".tar.gz"
}
