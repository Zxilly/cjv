package toolchain

import (
	"regexp"
	"strings"
	"time"

	goversion "github.com/hashicorp/go-version"
)

var minorSelector = regexp.MustCompile(`^[0-9]+\.[0-9]+$`)
var dateSelector = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
var nightlyTimestamp = regexp.MustCompile(`(?:^|\.)([0-9]{8})[0-9]{6}(?:$|-)`)
var stableVersion = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+$`)

func IsVersionSelector(version string) bool {
	return minorSelector.MatchString(version) || dateSelector.MatchString(version)
}

// MatchesVersionSelector resolves shorthand against published versions; it
// never synthesizes a download URL or assumes an unpublished archive exists.
func MatchesVersionSelector(selector, version string) bool {
	if minorSelector.MatchString(selector) {
		return strings.HasPrefix(version, selector+".") && stableVersion.MatchString(version)
	}
	if dateSelector.MatchString(selector) {
		date, err := time.Parse("2006-01-02", selector)
		if err != nil {
			return false
		}
		match := nightlyTimestamp.FindStringSubmatch(version)
		return len(match) > 1 && match[1] == date.Format("20060102")
	}
	return selector == version
}

// CompareVersions orders two version strings: -1, 0, or 1 as a is older,
// equal, or newer than b. Valid semantic versions compare numerically and
// sort above unparsable strings, which compare lexically.
func CompareVersions(a, b string) int {
	va, aErr := goversion.NewVersion(a)
	vb, bErr := goversion.NewVersion(b)
	switch {
	case aErr == nil && bErr == nil:
		return va.Compare(vb)
	case aErr == nil:
		return 1
	case bErr == nil:
		return -1
	default:
		return strings.Compare(a, b)
	}
}
