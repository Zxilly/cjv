package lifecycle

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/fsops"
	"github.com/Zxilly/cjv/internal/fstx"
)

type preparedHistory struct {
	entry         HistoryEntry
	home          string
	path, staging string
}

func historyRoots(base, name string) component.Roots {
	return component.Roots{TcDir: filepath.Join(base, "sdk", name), DocsDir: filepath.Join(base, "docs", name), StdxDir: filepath.Join(base, "stdx", name)}
}

func historyRootPaths(roots component.Roots) [][2]string {
	return [][2]string{{"sdk", roots.TcDir}, {"docs", roots.DocsDir}, {"stdx", roots.StdxDir}}
}

func prepareHistorySnapshot(ctx context.Context, home, owner, hostTuple string, members []*groupMember) (_ *preparedHistory, retErr error) {
	root, err := openHistoryDirectory(home, owner, true)
	if err != nil {
		return nil, err
	}
	defer root.Close() //nolint:errcheck
	now := time.Now().UTC()
	var suffix [6]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		return nil, err
	}
	id := now.Format(historyTimeFormat) + "-" + hex.EncodeToString(suffix[:])
	base := filepath.Join(home, "history", owner)
	snapshot := &preparedHistory{entry: HistoryEntry{FormatVersion: historyFormatVersion, ID: id, Toolchain: owner, HostTuple: hostTuple, CreatedAt: now, Members: []HistoryMember{}}, home: home, path: filepath.Join(base, id), staging: filepath.Join(base, id+".staging")}
	if err := root.Mkdir(id+".staging", 0o755); err != nil {
		return nil, err
	}
	defer func() {
		if retErr != nil {
			_ = fsops.RemoveOwnedTree(root, id+".staging")
		}
	}()
	for _, member := range members {
		if member.before == nil {
			continue
		}
		current, err := component.RootsFor(member.identity)
		if err != nil {
			return nil, err
		}
		stored := historyRoots(snapshot.staging, member.identity)
		materialize := make(map[string]bool)
		for _, intent := range member.original {
			if intent.Source == "" {
				continue
			}
			spec, err := component.SpecFor(intent.Name)
			if err != nil {
				return nil, err
			}
			for _, child := range spec.LinkChildren {
				materialize[filepath.Join(spec.InstallRoot(current), child)] = true
			}
		}
		for i, pair := range historyRootPaths(current) {
			if _, err := os.Lstat(pair[1]); errors.Is(err, os.ErrNotExist) && i > 0 {
				continue
			} else if err != nil {
				return nil, err
			}
			if err := copyHistoryTree(ctx, pair[1], historyRootPaths(stored)[i][1], materialize); err != nil {
				return nil, fmt.Errorf("snapshot %s %s: %w", member.identity, pair[0], err)
			}
		}
		// Linked component roots are now owned copies. Expand their manifests
		// so later component removal removes files, not a nonempty directory.
		for _, intent := range member.original {
			if intent.Source == "" {
				continue
			}
			spec, _ := component.SpecFor(intent.Name)
			var files []string
			for _, child := range spec.LinkChildren {
				base := spec.InstallRoot(stored)
				err := filepath.WalkDir(filepath.Join(base, child), func(path string, item fs.DirEntry, err error) error {
					if err != nil {
						return err
					}
					if item.IsDir() {
						return nil
					}
					rel, err := filepath.Rel(base, path)
					if err != nil {
						return err
					}
					files = append(files, filepath.ToSlash(rel))
					return nil
				})
				if err != nil {
					return nil, err
				}
			}
			if err := component.WriteManifest(stored.TcDir, intent.Name, files); err != nil {
				return nil, err
			}
		}
		record := member.before
		entry := HistoryMember{Name: member.identity, Release: record.Release, Tuple: record.Tuple, SHA256: record.SHA256, Roots: []HistoryRoot{}}
		for _, pair := range historyRootPaths(stored) {
			if _, err := os.Lstat(pair[1]); errors.Is(err, os.ErrNotExist) {
				continue
			} else if err != nil {
				return nil, err
			}
			digest, err := historyTreeDigest(ctx, pair[1])
			if err != nil {
				return nil, err
			}
			entry.Roots = append(entry.Roots, HistoryRoot{Kind: pair[0], SHA256: digest})
		}
		snapshot.entry.Members = append(snapshot.entry.Members, entry)
	}
	if err := validateHistoryEntry(snapshot.entry, owner, id); err != nil {
		return nil, err
	}
	data, err := json.MarshalIndent(snapshot.entry, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := fsops.WriteFileAtomic(filepath.Join(snapshot.staging, historyManifestName), append(data, '\n'), 0o644); err != nil {
		return nil, err
	}
	if err := syncHistoryDirectories(ctx, snapshot.staging); err != nil {
		return nil, err
	}
	return snapshot, nil
}

func syncHistoryDirectories(ctx context.Context, root string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	var dirs []string
	if err := filepath.WalkDir(root, func(path string, item fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if item.IsDir() {
			dirs = append(dirs, path)
		}
		return nil
	}); err != nil {
		return err
	}
	for i := len(dirs) - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			return err
		}
		f, err := os.Open(dirs[i])
		if err != nil {
			return err
		}
		if err := errors.Join(f.Sync(), f.Close()); err != nil {
			return err
		}
	}
	// Persist the newly created owner and history directories as well.
	dir := filepath.Dir(root)
	for range 3 {
		f, err := os.Open(dir)
		if err != nil {
			return err
		}
		if err := errors.Join(f.Sync(), f.Close()); err != nil {
			return err
		}
		dir = filepath.Dir(dir)
	}
	return nil
}

func (s *preparedHistory) cleanup(err error) {
	var recovery *fstx.RecoveryError
	if !errors.As(err, &recovery) {
		root, err := openHistoryDirectory(s.home, s.entry.Toolchain, false)
		if err == nil {
			_ = fsops.RemoveOwnedTree(root, filepath.Base(s.staging))
			_ = root.Close()
		}
	}
}

// copyHistoryTree always copies file bytes, including from hardlinked SDKs.
// Only registered component links may cross a source root; those become files.
func copyHistoryTree(ctx context.Context, source, dest string, materialize map[string]bool) error {
	info, err := os.Lstat(source)
	if err != nil {
		return err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("history content root must be a real directory: %s", source)
	}
	var directories []struct {
		path string
		mode os.FileMode
	}
	if err := filepath.WalkDir(source, func(path string, item fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		to := filepath.Join(dest, rel)
		info, err := item.Info()
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			if materialize[path] {
				resolved, err := filepath.EvalSymlinks(path)
				if err != nil {
					return err
				}
				return copyHistoryTree(ctx, resolved, to, nil)
			}
			link, err := safeHistoryLink(source, path)
			if err != nil {
				return err
			}
			return os.Symlink(link, to)
		}
		if item.IsDir() {
			// Populate read-only directories before restoring their permissions.
			// Chmod also cancels any permission bits removed by the caller's umask.
			if err := os.MkdirAll(to, 0o700); err != nil {
				return err
			}
			if err := os.Chmod(to, 0o700); err != nil {
				return err
			}
			directories = append(directories, struct {
				path string
				mode os.FileMode
			}{to, info.Mode()})
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported history file: %s", path)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		out, err := os.OpenFile(to, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			_ = in.Close()
			return err
		}
		_, copyErr := fsops.CopyContext(ctx, out, in)
		if copyErr == nil {
			copyErr = out.Chmod(info.Mode())
		}
		return errors.Join(copyErr, out.Sync(), out.Close(), in.Close())
	}); err != nil {
		return err
	}
	for i := len(directories) - 1; i >= 0; i-- {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := os.Chmod(directories[i].path, directories[i].mode); err != nil {
			return err
		}
	}
	return nil
}

func safeHistoryLink(root, path string) (string, error) {
	link, err := os.Readlink(path)
	if err != nil {
		return "", err
	}
	if filepath.IsAbs(link) {
		return "", fmt.Errorf("history link must be relative: %s", path)
	}
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	// Compare both paths after resolving ancestor links, such as /var on macOS.
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(resolvedRoot, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
		return "", fmt.Errorf("history link leaves its content root: %s", path)
	}
	return link, nil
}

func historyTreeDigest(ctx context.Context, root string) (string, error) {
	info, err := os.Lstat(root)
	if err != nil {
		return "", err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return "", fmt.Errorf("history content root must be a directory: %s", root)
	}
	hash := sha256.New()
	err = filepath.WalkDir(root, func(path string, item fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		info, err := item.Info()
		if err != nil {
			return err
		}
		_, _ = fmt.Fprintf(hash, "%s\x00%d\x00", filepath.ToSlash(rel), info.Mode())
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := safeHistoryLink(root, path)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(hash, "%s\x00", link)
			return nil
		}
		if item.IsDir() {
			return nil
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported history file: %s", path)
		}
		_, _ = fmt.Fprintf(hash, "%d\x00", info.Size())
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := fsops.CopyContext(ctx, hash, file)
		return errors.Join(copyErr, file.Close())
	})
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
