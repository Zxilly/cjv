package selfupdate

import (
	"context"

	"github.com/Zxilly/cjv/internal/progress"
)

// Status describes whether a build can be updated and whether it changed.
type Status string

const (
	StatusSkipped     Status = "skipped"
	StatusDevelopment Status = "dev"
	StatusUpToDate    Status = "up-to-date"
	StatusUpdated     Status = "updated"
)

// Result records the running version and the version installed after Update.
// A successful call does not imply that an update was available or applied.
type Result struct {
	CurrentVersion string
	Version        string
	Status         Status
}

// Update checks for and applies a self-update. The actual flow (GitHub vs
// GitCode) is selected at compile time via the `mirror` build tag — see
// update_default.go and update_mirror.go.
//
// updateURL is the releases URL embedded at build time; currentVersion is the
// running binary's version (or "dev" for unstamped local builds). The release
// asset download reports its progress to sink; nil reports nothing.
func Update(ctx context.Context, updateURL, currentVersion string, sink progress.Sink) (Result, error) {
	if updateURL == "" {
		return Result{CurrentVersion: currentVersion, Version: currentVersion, Status: StatusSkipped}, nil
	}
	if currentVersion == "dev" {
		return Result{CurrentVersion: currentVersion, Version: currentVersion, Status: StatusDevelopment}, nil
	}
	return runUpdate(ctx, updateURL, currentVersion, sink)
}
