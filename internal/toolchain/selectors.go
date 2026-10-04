package toolchain

import (
	"regexp"
	"strings"
	"time"
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
