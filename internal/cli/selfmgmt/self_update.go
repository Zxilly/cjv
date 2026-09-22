package selfmgmt

import (
	"context"
	"log/slog"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/env"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/proxy"
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
func UpdateManaged(ctx context.Context, updateURL, currentVersion string) (UpdateResult, error) {
	// Recover an interrupted update before bootstrapping a missing binary.
	selfupdate.CleanupOldBinaries()
	if _, err := selfupdate.EnsureManagedExecutable(); err != nil {
		return UpdateResult{}, err
	}
	result, err := selfupdate.Update(ctx, updateURL, currentVersion)
	if err != nil {
		return UpdateResult{}, err
	}
	outcome := UpdateResult{
		Version:         result.Version,
		Updated:         result.Status == selfupdate.StatusUpdated,
		Status:          result.Status,
		previousVersion: result.CurrentVersion,
	}
	if err := proxy.CreateAllProxyLinks(); err != nil {
		return outcome, &UpdateFinalizationError{Result: outcome, Err: err}
	}
	// Scripts are static and self-locating. Refreshing them is idempotent and
	// never edits shell profiles or registry PATH entries. A convenience-script
	// failure must not report a successfully replaced binary as a failed update.
	if err := refreshEnvScripts(); err != nil {
		slog.Warn("failed to refresh env scripts during self update", "error", err)
	}
	return outcome, nil
}

func refreshEnvScripts() error {
	home, err := config.Home()
	if err != nil {
		return err
	}
	binDir, err := config.BinDir()
	if err != nil {
		return err
	}
	if err := config.EnsureDirs(); err != nil {
		return err
	}
	return env.WriteEnvScripts(home, binDir)
}
