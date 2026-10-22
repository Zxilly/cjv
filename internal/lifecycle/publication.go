package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/fsops"
	"github.com/Zxilly/cjv/internal/fstx"
	"github.com/Zxilly/cjv/internal/progress"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// publishPrepared runs with the home lock, after all members have been staged
// and revalidated. Every root shares the same commit and recovery decision.
func publishPrepared(ctx context.Context, home string, identities []string, pairs [][2]string, publish func() error, opts Options) (retErr error) {
	root, err := os.OpenRoot(home)
	if err != nil {
		return err
	}
	defer root.Close() //nolint:errcheck
	removeStaging := func(destination string) error {
		rel, err := filepath.Rel(home, config.StagingDir(destination))
		if err != nil {
			return err
		}
		return fsops.RemoveOwnedTree(root, rel)
	}
	defer func() {
		var recovery *fstx.RecoveryError
		if !errors.As(retErr, &recovery) {
			for _, pair := range pairs {
				_ = removeStaging(pair[1])
			}
		}
	}()
	for _, pair := range pairs {
		if err := ctx.Err(); err != nil {
			return err
		}
		stage := config.StagingDir(pair[1])
		if err := removeStaging(pair[1]); err != nil {
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
	if err := ctx.Err(); err != nil {
		return err
	}
	transactionNames := identities
	if opts.history != nil && len(identities) > 0 {
		owner := opts.history.entry.Toolchain
		parsedOwner, err := toolchain.ParseToolchainName(owner)
		if err != nil || parsedOwner.IsCustom() || parsedOwner.Channel == toolchain.UnknownChannel || parsedOwner.Version != "" || parsedOwner.Target != "" {
			return fmt.Errorf("history snapshot owner must be a tracking host")
		}
		// A target-only replacement still publishes history owned by its
		// unchanged host. Include that owner in the journal's allowed scope,
		// without moving its SDK or emitting an installation event for it.
		transactionNames = []string{owner}
		for _, identity := range identities {
			if identity == owner {
				continue
			}
			name, err := toolchain.ParseToolchainName(identity)
			if err != nil || name.IsCustom() || name.Channel != parsedOwner.Channel || name.Version != "" || name.Host != "" || !strings.HasPrefix(name.Target, opts.history.entry.HostTuple+"-") {
				return fmt.Errorf("history snapshot owner does not match the published group")
			}
			transactionNames = append(transactionNames, identity)
		}
	}
	tx, err := fstx.NewToolchainGroupTransaction(home, transactionNames)
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			retErr = errors.Join(retErr, tx.Rollback())
		}
	}()
	if opts.history != nil {
		if err := tx.IncludeHistory(opts.history.entry.ID); err != nil {
			return err
		}
		if err := tx.RenameFile(opts.history.staging, opts.history.path); err != nil {
			return err
		}
	}
	for _, pair := range pairs {
		if err := ctx.Err(); err != nil {
			return err
		}
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
	if err := finalizeInstalledToolchain(ctx); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
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
