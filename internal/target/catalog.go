package target

import (
	"slices"
	"strings"
)

//go:generate go run ../../scripts/gen-platform-surfaces.go

// HostPlatform is a (GOOS, GOARCH) pair cjv ships a host binary for.
type HostPlatform struct {
	GOOS   string
	GOARCH string
}

// SupportedHostPlatforms returns the (GOOS, GOARCH) pairs cjv builds host
// binaries for, sorted for determinism. This is the single source of truth for
// the shipped platform set; generated release/web surfaces and the sync tests
// should flow from this catalogue.
func SupportedHostPlatforms() []HostPlatform {
	out := make([]HostPlatform, 0, len(hostByGo))
	for key := range hostByGo {
		goos, goarch, _ := strings.Cut(key, "-")
		out = append(out, HostPlatform{GOOS: goos, GOARCH: goarch})
	}
	slices.SortFunc(out, func(a, b HostPlatform) int {
		if a.GOOS != b.GOOS {
			return strings.Compare(a.GOOS, b.GOOS)
		}
		return strings.Compare(a.GOARCH, b.GOARCH)
	})
	return out
}

var (
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
