package lifecycle

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/fsops"
	"github.com/Zxilly/cjv/internal/fstx"
	"github.com/Zxilly/cjv/internal/progress"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// publishOfficial holds the home lock. The three roots share one journal and
// one commit marker, including the release record and component manifests.
func publishOfficial(prepared component.Roots, identity string, reinstall bool, publish func() error, opts Options) (retErr error) {
	home, err := config.Home()
	if err != nil {
		return err
	}
	dest, err := component.RootsFor(identity)
	if err != nil {
		return err
	}
	if reinstall && opts.prepare == nil {
		// A forced fixed-version reinstall preserves its external components.
		if err := preserveComponentMetadata(prepared.TcDir, dest.TcDir); err != nil {
			return err
		}
		for _, pair := range [][2]string{{dest.DocsDir, prepared.DocsDir}, {dest.StdxDir, prepared.StdxDir}} {
			if _, err := os.Lstat(pair[0]); errors.Is(err, os.ErrNotExist) {
				continue
			} else if err != nil {
				return err
			}
			if err := fsops.CopyTree(pair[0], pair[1]); err != nil {
				return err
			}
		}
	}
	_, settings, err := config.LoadDefaultSettings()
	if err != nil {
		return err
	}
	if settings.LinkMode == "hardlink" {
		if err := deduplicateSDK(prepared.TcDir, identity); err != nil {
			return err
		}
	}
	pairs := [][2]string{{prepared.TcDir, dest.TcDir}, {prepared.DocsDir, dest.DocsDir}, {prepared.StdxDir, dest.StdxDir}}
	return publishPrepared(home, []string{identity}, pairs, publish, opts)
}

// publishPrepared runs with the home lock, after all members have been staged
// and revalidated. Every root shares the same commit and recovery decision.
func publishPrepared(home string, identities []string, pairs [][2]string, publish func() error, opts Options) (retErr error) {
	defer func() {
		var recovery *fstx.RecoveryError
		if !errors.As(retErr, &recovery) {
			for _, pair := range pairs {
				_ = fsops.RemoveAllRetry(config.StagingDir(pair[1]))
			}
		}
	}()
	for _, pair := range pairs {
		stage := config.StagingDir(pair[1])
		if err := fsops.RemoveAllRetry(stage); err != nil {
			return err
		}
		if _, err := os.Lstat(pair[0]); err == nil {
			if err := fsops.RenameRetry(pair[0], stage); err != nil {
				if err := fsops.CopyTree(pair[0], stage); err != nil {
					return err
				}
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if afterStagingHook != nil {
		if err := afterStagingHook(); err != nil {
			return err
		}
	}
	tx, err := fstx.NewToolchainGroupTransaction(home, identities)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			retErr = errors.Join(retErr, tx.Rollback())
		}
	}()
	for _, pair := range pairs {
		if _, err := os.Lstat(pair[1]); err == nil {
			if err := tx.RemoveDir(pair[1]); err != nil {
				return err
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		stage := config.StagingDir(pair[1])
		if _, err := os.Lstat(stage); err == nil {
			if err := tx.RenameFile(stage, pair[1]); err != nil {
				return err
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	if err := finalizeInstalledToolchain(); err != nil {
		return err
	}
	publicationErr, err := commitInstallation(tx, publish)
	if err != nil {
		return err
	}
	committed = true
	for i, identity := range identities {
		if pairs[i*3][0] != "" {
			opts.emit(progress.Event{Kind: progress.ToolchainInstalled, Toolchain: identity})
		}
	}
	return publicationErr
}

// publishedSettingsError means restoring references failed. Retain the usable
// installation that persisted settings may still select, and report the error.
type publishedSettingsError struct{ error }

func (e *publishedSettingsError) Unwrap() error { return e.error }

func commitInstallation(tx *fstx.Transaction, publish func() error) (publicationErr, commitErr error) {
	if publish == nil {
		return nil, tx.Commit()
	}
	commitErr = tx.CommitWith(func() error {
		err := publish()
		var retained *publishedSettingsError
		if errors.As(err, &retained) {
			publicationErr = err
			return nil
		}
		return err
	})
	return publicationErr, commitErr
}

func deduplicateSDK(prepared, identity string) error {
	// prepared's name is scratch, so decode the record directly instead of
	// applying installed-directory identity validation.
	record, err := readPreparedRecord(prepared)
	if err != nil {
		return err
	}
	if record.SHA256 == "" {
		return nil
	}
	installed, err := toolchain.ListInstalled()
	if err != nil {
		return err
	}
	for _, candidate := range installed {
		if candidate == identity {
			continue
		}
		dir, err := config.ToolchainDirFor(candidate)
		if err != nil {
			return err
		}
		other, err := toolchain.ReadInstallation(dir)
		if err != nil || other != record {
			continue
		}
		// Compare bytes as well as provenance: a user may have changed an
		// installed SDK. Never substitute those changes into a verified stage.
		if err := fsops.HardlinkIdenticalTree(dir, prepared); err != nil {
			return fmt.Errorf("link SDK payload: %w", err)
		}
		break
	}
	return nil
}

func readPreparedRecord(dir string) (toolchain.Installation, error) {
	var record toolchain.Installation
	_, err := toml.DecodeFile(filepath.Join(dir, ".cjv", "toolchain.toml"), &record)
	return record, err
}
