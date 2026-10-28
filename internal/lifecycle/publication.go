package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/fsops"
	"github.com/Zxilly/cjv/internal/fstx"
	"github.com/Zxilly/cjv/internal/progress"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// rootChange replaces a destination with prepared content. An empty source
// explicitly removes the destination without installing a replacement.
type rootChange struct {
	source      string
	destination string
}

// toolchainPublication keeps each installation identity with its root changes.
// Component-only changes need not replace the SDK or every component root.
type toolchainPublication struct {
	identity string
	roots    []rootChange
	removed  bool
}

func toolchainRoots(source, destination component.Roots) []rootChange {
	return []rootChange{
		{source.TcDir, destination.TcDir},
		{source.DocsDir, destination.DocsDir},
		{source.StdxDir, destination.StdxDir},
	}
}

// rootPublication owns one durable commit decision, including rollback when
// placement, finalization or settings publication fails. Callers choose the
// transaction scope and retain their staging until recovery has succeeded.
type rootPublication struct {
	roots      []rootChange
	history    *preparedHistory
	finalize   func() error
	publish    func() error
	afterRoot  func(string) error
	createOnly bool
}

func (p rootPublication) commit(ctx context.Context, tx *fstx.Transaction) (committed bool, retErr error) {
	defer func() {
		if !committed {
			if err := tx.Rollback(); err != nil {
				retErr = errors.Join(retErr, fmt.Errorf("rollback publication: %w", err))
			}
		}
	}()
	if p.history != nil {
		if err := tx.IncludeHistory(p.history.entry.ID); err != nil {
			return false, err
		}
		if err := tx.RenameFile(p.history.staging, p.history.path); err != nil {
			return false, err
		}
	}
	for _, change := range p.roots {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if !p.createOnly {
			if _, err := os.Lstat(change.destination); err == nil {
				if err := tx.RemoveDir(change.destination); err != nil {
					return false, fmt.Errorf("remove existing %s: %w", change.destination, err)
				}
			} else if !errors.Is(err, os.ErrNotExist) {
				return false, err
			}
		}
		if change.source != "" {
			if err := tx.RenameFile(change.source, change.destination); err != nil {
				return false, fmt.Errorf("place %s: %w", change.destination, err)
			}
		}
		if p.afterRoot != nil {
			if err := p.afterRoot(change.destination); err != nil {
				return false, err
			}
		}
	}
	if p.finalize != nil {
		if err := p.finalize(); err != nil {
			return false, err
		}
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	publicationErr, err := commitInstallation(tx, p.publish)
	if err != nil {
		return false, err
	}
	committed = true
	return true, publicationErr
}

// publishPrepared runs with the home lock, after all members have been staged
// and revalidated. Every root shares the same commit and recovery decision.
func publishPrepared(ctx context.Context, home string, members []toolchainPublication, publish func() error, opts Options) (retErr error) {
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
			for _, member := range members {
				for _, change := range member.roots {
					_ = removeStaging(change.destination)
				}
			}
		}
	}()
	var identities []string
	publication := rootPublication{history: opts.history, publish: publish, finalize: func() error {
		return finalizeInstalledToolchain(ctx)
	}}
	for _, member := range members {
		identities = append(identities, member.identity)
		for _, change := range member.roots {
			if err := ctx.Err(); err != nil {
				return err
			}
			stage := config.StagingDir(change.destination)
			if err := removeStaging(change.destination); err != nil {
				return err
			}
			staged := rootChange{destination: change.destination}
			if change.source != "" {
				if _, err := os.Lstat(change.source); err == nil {
					if err := fsops.RenameRetry(change.source, stage); err != nil {
						if err := fsops.CopyTree(change.source, stage); err != nil {
							return err
						}
					}
					staged.source = stage
				} else if !errors.Is(err, os.ErrNotExist) {
					return err
				}
			}
			publication.roots = append(publication.roots, staged)
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
	transactionNames, err := publicationNames(identities, opts.history)
	if err != nil {
		return err
	}
	tx, err := fstx.NewToolchainGroupTransaction(home, transactionNames)
	if err != nil {
		return err
	}
	committed, publicationErr := publication.commit(ctx, tx)
	if !committed {
		return publicationErr
	}
	for _, member := range members {
		if !member.removed {
			opts.emit(progress.Event{Kind: progress.ToolchainInstalled, Toolchain: member.identity})
		}
	}
	return publicationErr
}

func publicationNames(identities []string, history *preparedHistory) ([]string, error) {
	if history == nil || len(identities) == 0 {
		return identities, nil
	}
	owner := history.entry.Toolchain
	parsedOwner, err := toolchain.ParseToolchainName(owner)
	if err != nil || parsedOwner.IsCustom() || parsedOwner.Channel == toolchain.UnknownChannel || parsedOwner.Version != "" || parsedOwner.Target != "" {
		return nil, fmt.Errorf("history snapshot owner must be a tracking host")
	}
	// An unchanged host still owns history for target-only replacements.
	names := []string{owner}
	for _, identity := range identities {
		if identity == owner {
			continue
		}
		name, err := toolchain.ParseToolchainName(identity)
		if err != nil || name.IsCustom() || name.Channel != parsedOwner.Channel || name.Version != "" || name.Host != "" || !strings.HasPrefix(name.Target, history.entry.HostTuple+"-") {
			return nil, fmt.Errorf("history snapshot owner does not match the published group")
		}
		names = append(names, identity)
	}
	return names, nil
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
