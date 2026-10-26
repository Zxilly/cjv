package lifecycle

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/fsops"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
)

const (
	DefaultHistoryRetention = 2
	historyFormatVersion    = 1
	historyManifestName     = "snapshot.json"
	historyTimeFormat       = "20060102T150405.000000000Z"
)

// HistoryEntry describes one independent, complete previous tracking group.
// Root paths are derived from validated kinds and names, never read from JSON.
type HistoryEntry struct {
	FormatVersion int             `json:"format_version"`
	ID            string          `json:"id"`
	Toolchain     string          `json:"toolchain"`
	HostTuple     string          `json:"host_tuple"`
	CreatedAt     time.Time       `json:"created_at"`
	Members       []HistoryMember `json:"members"`
}

type HistoryMember struct {
	Name    string        `json:"name"`
	Release string        `json:"release"`
	Tuple   string        `json:"tuple,omitempty"`
	SHA256  string        `json:"sha256,omitempty"`
	Roots   []HistoryRoot `json:"roots"`
}

type HistoryRoot struct {
	Kind   string `json:"kind"`
	SHA256 string `json:"sha256"`
}

// InvalidHistory identifies an owned snapshot whose metadata cannot be read.
// Payload validation remains mandatory when a snapshot is selected for rollback.
type InvalidHistory struct {
	Toolchain string `json:"toolchain"`
	ID        string `json:"id"`
	Error     string `json:"error"`
}

type HistoryReport struct {
	Snapshots []HistoryEntry   `json:"snapshots"`
	Invalid   []InvalidHistory `json:"invalid_snapshots,omitempty"`
}

func (m HistoryMember) installation() toolchain.Installation {
	return toolchain.Installation{Release: m.Release, Tuple: m.Tuple, SHA256: m.SHA256}
}

// ListHistory returns readable snapshot metadata, newest first within each
// tracking group. InspectHistory also reports damaged metadata. Both are
// read-only, including for an absent home and legacy layouts.
func ListHistory(ctx context.Context, name string) ([]HistoryEntry, error) {
	report, err := InspectHistory(ctx, name)
	return report.Snapshots, err
}

// InspectHistory keeps one damaged snapshot from hiding other usable entries.
// Errors accessing the history directory itself still fail the inspection.
func InspectHistory(ctx context.Context, name string) (HistoryReport, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	home, err := config.Home()
	if err != nil {
		return HistoryReport{}, err
	}
	if name != "" {
		name, err = historyIdentity(name)
		if err != nil {
			return HistoryReport{}, err
		}
	}
	return inspectHistoryAt(ctx, home, name)
}

func historyIdentity(input string) (string, error) {
	name, err := toolchain.ParseToolchainName(input)
	if err != nil || name.IsCustom() || name.Channel == toolchain.UnknownChannel || name.Version != "" || name.Target != "" {
		return "", fmt.Errorf("history requires a tracking host toolchain, got %q", input)
	}
	if dir, err := toolchain.FindInstalled(name); err == nil {
		return filepath.Base(dir), nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", err
	}
	return name.String(), nil
}

func inspectHistoryAt(ctx context.Context, home, name string) (HistoryReport, error) {
	result := HistoryReport{Snapshots: []HistoryEntry{}}
	root, err := openHistoryDirectory(home, "", false)
	if errors.Is(err, os.ErrNotExist) {
		return result, nil
	}
	if err != nil {
		return result, err
	}
	defer root.Close() //nolint:errcheck
	owners := []string{name}
	if name == "" {
		owners = nil
		entries, err := readRootDirectory(root)
		if err != nil {
			return result, err
		}
		for _, entry := range entries {
			parsed, parseErr := toolchain.ParseToolchainName(entry.Name())
			if parseErr == nil && !parsed.IsCustom() && parsed.Channel != toolchain.UnknownChannel && parsed.Version == "" && parsed.Target == "" {
				owners = append(owners, entry.Name())
			}
		}
		slices.Sort(owners)
	}
	for _, owner := range owners {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		dir, err := openHistoryDirectory(home, owner, false)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return result, err
		}
		entries, readErr := readRootDirectory(dir)
		closeErr := dir.Close()
		if err := errors.Join(readErr, closeErr); err != nil {
			return result, err
		}
		var group []HistoryEntry
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return result, err
			}
			if !config.IsHistoryID(entry.Name()) {
				continue
			}
			snapshot, err := readHistoryEntry(home, owner, entry.Name())
			if err != nil {
				result.Invalid = append(result.Invalid, InvalidHistory{Toolchain: owner, ID: entry.Name(), Error: err.Error()})
				continue
			}
			group = append(group, snapshot)
		}
		slices.SortFunc(group, func(a, b HistoryEntry) int { return strings.Compare(b.ID, a.ID) })
		result.Snapshots = append(result.Snapshots, group...)
	}
	slices.SortFunc(result.Invalid, func(a, b InvalidHistory) int {
		if a.Toolchain != b.Toolchain {
			return strings.Compare(a.Toolchain, b.Toolchain)
		}
		return strings.Compare(b.ID, a.ID)
	})
	return result, nil
}

// PruneHistory keeps the newest keep entries per selected tracking host. A zero
// retention also removes damaged snapshot entries; installed SDKs and fixed
// versions stay put. Nonzero retention preserves damaged entries for inspection
// without letting them block pruning healthy snapshots.
func PruneHistory(ctx context.Context, name string, keep int) (int, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if keep < 0 {
		return 0, errors.New("history retention must be nonnegative")
	}
	if name != "" {
		var err error
		name, err = historyIdentity(name)
		if err != nil {
			return 0, err
		}
	}
	home, err := config.Home()
	if err != nil {
		return 0, err
	}
	if _, err := os.Stat(home); errors.Is(err, os.ErrNotExist) {
		return 0, nil
	} else if err != nil {
		return 0, err
	}
	lock, err := toolchain.LockHome(ctx)
	if err != nil {
		return 0, err
	}
	defer lock.Close() //nolint:errcheck
	if err := lock.Recover(); err != nil {
		return 0, err
	}
	removed := 0
	err = lock.WithStableReads(ctx, func() error {
		var err error
		removed, err = pruneHistoryLocked(ctx, home, name, keep)
		return err
	})
	return removed, err
}

func pruneHistoryLocked(ctx context.Context, home, name string, keep int) (int, error) {
	report, err := inspectHistoryAt(ctx, home, name)
	if err != nil {
		return 0, err
	}
	entries := report.Snapshots
	if keep == 0 {
		for _, invalid := range report.Invalid {
			entries = append(entries, HistoryEntry{Toolchain: invalid.Toolchain, ID: invalid.ID})
		}
	}
	seen := make(map[string]int)
	removed := 0
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return removed, err
		}
		seen[entry.Toolchain]++
		if seen[entry.Toolchain] <= keep {
			continue
		}
		root, err := openHistoryDirectory(home, entry.Toolchain, false)
		if err != nil {
			return removed, err
		}
		// Stop advertising the snapshot before deleting any of its payload.
		// Interrupted cleanup is resumed as ordinary history staging residue.
		staging := entry.ID + config.StagingSuffix
		_, statErr := root.Lstat(staging)
		var removeErr error
		if !errors.Is(statErr, os.ErrNotExist) {
			removeErr = statErr
			if removeErr == nil {
				removeErr = fmt.Errorf("history cleanup path %s already exists", staging)
			}
		} else if err := fsops.Retry(func() error { return root.Rename(entry.ID, staging) }); err != nil {
			removeErr = err
		} else {
			if runtime.GOOS != "windows" {
				// Persist the hidden name before payload deletion can persist.
				var directory *os.File
				directory, removeErr = root.Open(".")
				if removeErr == nil {
					removeErr = errors.Join(directory.Sync(), directory.Close())
				}
			}
			if removeErr == nil {
				removeErr = fsops.RemoveOwnedTree(root, staging)
			}
		}
		if err := errors.Join(removeErr, root.Close()); err != nil {
			return removed, err
		}
		removed++
	}
	return removed, nil
}

func openHistoryDirectory(home, owner string, create bool) (*os.Root, error) {
	root, err := os.OpenRoot(home)
	if err != nil {
		return nil, err
	}
	parts := []string{config.HistorySubdir}
	if owner != "" {
		parsed, err := toolchain.ParseToolchainName(owner)
		if err != nil || parsed.IsCustom() || parsed.Channel == toolchain.UnknownChannel || parsed.Version != "" || parsed.Target != "" || parsed.String() != owner {
			_ = root.Close()
			return nil, fmt.Errorf("invalid history owner %q", owner)
		}
		parts = append(parts, owner)
	}
	for _, part := range parts {
		if create {
			if err := root.Mkdir(part, 0o755); err != nil && !errors.Is(err, os.ErrExist) {
				_ = root.Close()
				return nil, err
			}
		}
		info, err := root.Lstat(part)
		if err == nil && (!info.IsDir() || info.Mode()&os.ModeSymlink != 0) {
			err = fmt.Errorf("history directory %s must not be a file or link", part)
		}
		if err != nil {
			_ = root.Close()
			return nil, err
		}
		next, err := root.OpenRoot(part)
		_ = root.Close()
		if err != nil {
			return nil, err
		}
		root = next
	}
	return root, nil
}

func readRootDirectory(root *os.Root) ([]os.DirEntry, error) {
	f, err := root.Open(".")
	if err != nil {
		return nil, err
	}
	entries, err := f.ReadDir(-1)
	return entries, errors.Join(err, f.Close())
}

func readHistoryEntry(home, owner, id string) (HistoryEntry, error) {
	var entry HistoryEntry
	if !config.IsHistoryID(id) {
		return entry, fmt.Errorf("invalid history snapshot identifier %q", id)
	}
	root, err := openHistoryDirectory(home, owner, false)
	if err != nil {
		return entry, err
	}
	defer root.Close() //nolint:errcheck
	info, err := root.Lstat(id)
	if err != nil {
		return entry, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return entry, errors.New("history snapshot must be a directory, not a link")
	}
	snapshot, err := root.OpenRoot(id)
	if err != nil {
		return entry, err
	}
	defer snapshot.Close() //nolint:errcheck
	info, err = snapshot.Lstat(historyManifestName)
	if err != nil {
		return entry, err
	}
	if !info.Mode().IsRegular() || info.Size() > 1<<20 {
		return entry, errors.New("history metadata must be a regular file no larger than 1 MiB")
	}
	data, err := snapshot.ReadFile(historyManifestName)
	if err != nil {
		return entry, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&entry); err != nil {
		return entry, err
	}
	if err := decoder.Decode(new(any)); err != io.EOF {
		return entry, errors.New("history metadata contains trailing data")
	}
	return entry, validateHistoryEntry(entry, owner, id)
}

func validateHistoryEntry(entry HistoryEntry, owner, id string) error {
	if entry.FormatVersion != historyFormatVersion || entry.ID != id || entry.Toolchain != owner || entry.CreatedAt.IsZero() || !strings.HasPrefix(id, entry.CreatedAt.UTC().Format(historyTimeFormat)+"-") {
		return errors.New("invalid history metadata version, identity or timestamp")
	}
	host, err := sdktarget.ParseIdentity(entry.HostTuple)
	if err != nil || host.IsTargetVariant() || host.Tuple() != entry.HostTuple {
		return errors.New("invalid history host platform")
	}
	if len(entry.Members) == 0 || len(entry.Members) > 128 || entry.Members[0].Name != owner {
		return errors.New("history must contain its tracking host first")
	}
	ownerName, _ := toolchain.ParseToolchainName(owner)
	if ownerName.Host != "" && ownerName.Host != entry.HostTuple {
		return errors.New("history owner has an inconsistent platform")
	}
	seen := make(map[string]bool)
	for i, member := range entry.Members {
		name, nameErr := toolchain.ParseToolchainName(member.Name)
		release, releaseErr := toolchain.ParseToolchainName(member.Release)
		if nameErr != nil || releaseErr != nil || name.String() != member.Name || name.IsCustom() || name.Version != "" || name.Channel != ownerName.Channel || seen[member.Name] || release.IsCustom() || release.Version == "" || release.Channel != name.Channel || release.Host != name.Host || release.Target != name.Target {
			return fmt.Errorf("invalid history member %q", member.Name)
		}
		seen[member.Name] = true
		tuple := entry.HostTuple
		if i > 0 {
			if name.Host != "" || !strings.HasPrefix(name.Target, entry.HostTuple+"-") {
				return fmt.Errorf("history member %s belongs to a different host", member.Name)
			}
			tuple = name.Target
		}
		if member.Tuple != "" && member.Tuple != tuple {
			return fmt.Errorf("history member %s has an inconsistent platform", member.Name)
		}
		if member.SHA256 != "" && !validHistoryArtifactChecksum(member.SHA256) {
			return errors.New("invalid history artifact checksum")
		}
		kinds := make(map[string]bool)
		for _, root := range member.Roots {
			if root.Kind != "sdk" && root.Kind != "docs" && root.Kind != "stdx" || kinds[root.Kind] || !validHistoryDigest(root.SHA256) {
				return errors.New("invalid or duplicate history content root")
			}
			kinds[root.Kind] = true
		}
		if !kinds["sdk"] {
			return errors.New("history member has no SDK payload")
		}
	}
	return nil
}

func validHistoryDigest(value string) bool {
	return validHistoryArtifactChecksum(value) && strings.ToLower(value) == value
}

// Older installations retain the manifest's original checksum casing. Preserve
// it in snapshots so the stored installation record remains byte-for-byte
// consistent; locally generated content-tree digests stay canonical lowercase.
func validHistoryArtifactChecksum(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == 32
}
