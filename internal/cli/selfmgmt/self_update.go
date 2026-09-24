package selfmgmt

import (
	"context"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/progress"
	"github.com/Zxilly/cjv/internal/reachable"
	"github.com/Zxilly/cjv/internal/selfupdate"
)

// UpdateResult describes the managed installation after an update attempt.
type UpdateResult struct {
	Version string            `json:"version"`
	Updated bool              `json:"updated"`
	Status  selfupdate.Status `json:"status"`

	previousVersion string
}

// UpdateFinalizationError retains the installed version when refreshing its
// proxy links fails after the update attempt. The executable is not rolled back.
type UpdateFinalizationError struct {
	Result UpdateResult
	Err    error
}

func (e *UpdateFinalizationError) Error() string {
	return e.Result.Text() + "\n" + i18n.T("SelfUpdateProxyRefreshFailed", i18n.MsgData{"Err": e.Err.Error()})
}

func (e *UpdateFinalizationError) Unwrap() error { return e.Err }

func (e *UpdateFinalizationError) Code() cjverr.ErrorCode {
	return cjverr.ErrorCodeSelfUpdateFinalizationFailed
}

func (e *UpdateFinalizationError) Details() map[string]any {
	return map[string]any{
		"version": e.Result.Version,
		"updated": e.Result.Updated,
		"status":  e.Result.Status,
		"phase":   "proxy-refresh",
	}
}

func (r UpdateResult) Text() string {
	switch r.Status {
	case selfupdate.StatusSkipped:
		return i18n.T("MirrorNoAutoUpdate", nil)
	case selfupdate.StatusUpdated:
		return i18n.T("UpdateFound", i18n.MsgData{
			"Current": r.previousVersion,
			"Latest":  r.Version,
		}) + "\n" + i18n.T("UpdateApplied", i18n.MsgData{"Version": r.Version})
	default:
		return i18n.T("AlreadyUpToDate", i18n.MsgData{"Version": r.Version})
	}
}

// UpdateManaged updates the managed binary and refreshes its proxy links and
// shell scripts. Both explicit and automatic updates use this operation;
// callers decide how to render its result and whether failures are fatal.
// The release download reports its progress to sink; nil reports nothing.
func UpdateManaged(ctx context.Context, updateURL, currentVersion string, sink progress.Sink) (UpdateResult, error) {
	// Recover an interrupted update before bootstrapping a missing binary.
	selfupdate.CleanupOldBinaries()
	if _, err := selfupdate.EnsureManagedExecutable(); err != nil {
		return UpdateResult{}, err
	}
	result, err := selfupdate.Update(ctx, updateURL, currentVersion, sink)
	if err != nil {
		return UpdateResult{}, err
	}
	outcome := UpdateResult{
		Version:         result.Version,
		Updated:         result.Status == selfupdate.StatusUpdated,
		Status:          result.Status,
		previousVersion: result.CurrentVersion,
	}
	// The replaced binary must stay reachable: the proxy links point at it and
	// the env scripts are refreshed. Refreshing never edits shell profiles or
	// registry PATH entries, and a script failure only logs, so a successfully
	// replaced binary is never reported as a failed update.
	if err := reachable.Ensure(reachable.Policy{EnvScripts: true}); err != nil {
		return outcome, &UpdateFinalizationError{Result: outcome, Err: err}
	}
	return outcome, nil
}
