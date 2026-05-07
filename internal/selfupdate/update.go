package selfupdate

import (
	"context"
	"fmt"

	"github.com/Zxilly/cjv/internal/i18n"
)

// Update checks for and applies a self-update. The actual flow (GitHub vs
// GitCode) is selected at compile time via the `mirror` build tag — see
// update_default.go and update_mirror.go.
//
// updateURL is the releases URL embedded at build time; currentVersion is the
// running binary's version (or "dev" for unstamped local builds).
func Update(ctx context.Context, updateURL, currentVersion string) error {
	if updateURL == "" {
		fmt.Println(i18n.T("MirrorNoAutoUpdate", nil))
		return nil
	}
	if currentVersion == "dev" {
		fmt.Println(i18n.T("AlreadyUpToDate", i18n.MsgData{"Version": currentVersion}))
		return nil
	}
	return runUpdate(ctx, updateURL, currentVersion)
}
