package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"slices"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/toolchain"
)

type RollbackResult struct {
	SchemaVersion    int      `json:"schema_version"`
	Toolchain        string   `json:"toolchain"`
	Snapshot         string   `json:"snapshot"`
	Members          []string `json:"members"`
	PreviousSnapshot string   `json:"previous_snapshot,omitempty"`
}

// RollbackHistory restores a complete tracking group without network access.
// Its current payload is retained as another snapshot before publication.
func RollbackHistory(ctx context.Context, name, id string, opts Options) (RollbackResult, error) {
	result := RollbackResult{SchemaVersion: 1, Members: []string{}}
	if ctx == nil {
		ctx = context.Background()
	}
	owner, err := historyIdentity(name)
	if err != nil {
		return result, err
	}
	home, err := config.Home()
	if err != nil {
		return result, err
	}
	downloads, err := config.DownloadsDir()
	if err != nil {
		return result, err
	}
	preparation, err := dist.BeginPreparation(ctx, downloads)
	if err != nil {
		return result, err
	}
	defer preparation.Close() //nolint:errcheck
	lock, err := toolchain.LockHome(ctx)
	if err != nil {
		return result, err
	}
	defer lock.Close() //nolint:errcheck
	if err := lock.Recover(); err != nil {
		return result, err
	}
	if id == "" {
		entries, err := listHistoryAt(ctx, home, owner)
		if err != nil {
			return result, err
		}
		if len(entries) == 0 {
			return result, fmt.Errorf("no history for toolchain %s", owner)
		}
		id = entries[0].ID
	}
	entry, err := readHistoryEntry(home, owner, id)
	if err != nil {
		return result, err
	}
	base := filepath.Join(home, config.HistorySubdir, owner, id)
	if err := verifyHistoryPayload(ctx, base, entry); err != nil {
		return result, err
	}
	stage, err := preparation.TempDir(".cjv-rollback-*")
	if err != nil {
		return result, err
	}
	for _, member := range entry.Members {
		from, to := historyRoots(base, member.Name), historyRoots(stage, member.Name)
		for i, pair := range historyRootPaths(from) {
			if !slices.ContainsFunc(member.Roots, func(root HistoryRoot) bool { return root.Kind == pair[0] }) {
				continue
			}
			if err := copyHistoryTree(ctx, pair[1], historyRootPaths(to)[i][1], nil); err != nil {
				return result, err
			}
		}
	}
	// Verify the independent copies as well, before touching installed roots.
	if err := verifyHistoryPayload(ctx, stage, entry); err != nil {
		return result, err
	}
	parsed, _ := toolchain.ParseToolchainName(owner)
	tracked, err := trackedGroupTargets(parsed.Channel, entry.HostTuple, true)
	if err != nil {
		return result, err
	}
	currentNames := append([]string{owner}, tracked...)
	var current []*groupMember
	for _, currentName := range currentNames {
		roots, err := component.RootsFor(currentName)
		if err != nil {
			return result, err
		}
		record, err := toolchain.ReadInstallation(roots.TcDir)
		if errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return result, err
		}
		intents, err := component.InstalledIntents(roots)
		if err != nil {
			return result, err
		}
		current = append(current, &groupMember{identity: currentName, before: &record, original: intents})
	}
	if len(current) > 0 && current[0].identity == owner {
		snapshot, err := prepareHistorySnapshot(ctx, home, owner, entry.HostTuple, current)
		if err != nil {
			return result, err
		}
		opts.history = snapshot
		result.PreviousSnapshot = snapshot.entry.ID
	}
	identities := make([]string, 0, len(entry.Members)+len(tracked))
	var pairs [][2]string
	for _, member := range entry.Members {
		identities = append(identities, member.Name)
		from := historyRoots(stage, member.Name)
		to, err := component.RootsFor(member.Name)
		if err != nil {
			return result, err
		}
		for i, pair := range historyRootPaths(from) {
			pairs = append(pairs, [2]string{pair[1], historyRootPaths(to)[i][1]})
		}
	}
	for _, currentName := range tracked {
		if slices.Contains(identities, currentName) {
			continue
		}
		identities = append(identities, currentName)
		to, err := component.RootsFor(currentName)
		if err != nil {
			return result, err
		}
		for _, pair := range historyRootPaths(to) {
			pairs = append(pairs, [2]string{"", pair[1]})
		}
	}
	// Channel references stay attached to the same identity. Fixed versions
	// and other hosts never enter this transaction.
	publishErr := publishPrepared(ctx, home, identities, pairs, nil, opts)
	if opts.history != nil {
		opts.history.cleanup(publishErr)
	}
	if publishErr != nil {
		return result, publishErr
	}
	preparation.Complete()
	if _, err := pruneHistoryLocked(ctx, home, owner, DefaultHistoryRetention); err != nil {
		slog.Warn("could not prune toolchain history", "toolchain", owner, "error", err)
	}
	result.Toolchain, result.Snapshot = owner, id
	for _, member := range entry.Members {
		result.Members = append(result.Members, member.Name)
	}
	return result, nil
}

func verifyHistoryPayload(ctx context.Context, base string, entry HistoryEntry) error {
	snapshot, err := os.OpenRoot(base)
	if err != nil {
		return err
	}
	defer snapshot.Close() //nolint:errcheck
	for _, member := range entry.Members {
		roots := historyRoots(base, member.Name)
		for _, root := range member.Roots {
			// Deriving paths from metadata is not enough: reject links in the
			// intermediate kind directory before traversing the payload.
			info, err := snapshot.Lstat(root.Kind)
			if err != nil {
				return err
			}
			if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("history content directory %s must not be a link", root.Kind)
			}
			var path string
			for _, pair := range historyRootPaths(roots) {
				if pair[0] == root.Kind {
					path = pair[1]
				}
			}
			digest, err := historyTreeDigest(ctx, path)
			if err != nil {
				return err
			}
			if digest != root.SHA256 {
				return fmt.Errorf("history checksum mismatch for %s %s", member.Name, root.Kind)
			}
		}
		record, err := toolchain.ReadInstallation(roots.TcDir)
		if err != nil {
			return err
		}
		if record != member.installation() {
			return fmt.Errorf("history installation record mismatch for %s", member.Name)
		}
		tuple := member.Tuple
		if tuple == "" {
			name, _ := toolchain.ParseToolchainName(member.Name)
			tuple = name.Target
			if tuple == "" {
				tuple = entry.HostTuple
			}
		}
		if err := validateInstallation(roots.TcDir, tuple); err != nil {
			return fmt.Errorf("history %s: %w", member.Name, err)
		}
		if _, err := component.InstalledIntents(roots); err != nil {
			return err
		}
	}
	return nil
}
