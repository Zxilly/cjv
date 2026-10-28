package lifecycle

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// installationSnapshot contains observed state only. Reading one never locks,
// recovers, migrates or creates directories, so update inspection can use the
// same observation as mutation planning without acquiring write side effects.
type installationSnapshot struct {
	roots    component.Roots
	record   *toolchain.Installation
	intents  []component.Intent
	metadata string
}

func readInstallationSnapshot(identity string) (installationSnapshot, error) {
	roots, err := component.RootsFor(identity)
	if err != nil {
		return installationSnapshot{}, err
	}
	snapshot := installationSnapshot{roots: roots}
	record, err := toolchain.ReadInstallation(roots.TcDir)
	if errors.Is(err, os.ErrNotExist) {
		return snapshot, nil
	}
	if err != nil {
		return snapshot, err
	}
	snapshot.record = &record
	snapshot.intents, err = component.InstalledIntents(roots)
	return snapshot, err
}

// captureMetadata records the file contents that a writer must revalidate on
// returning from unlocked preparation. Read-only plans need only the intents.
func (s *installationSnapshot) captureMetadata() error {
	if s.record == nil {
		return nil
	}
	var err error
	s.metadata, err = componentState(filepath.Join(s.roots.TcDir, component.MetaDir))
	return err
}

// matches requires HomeLock. It rejects changes to the release, manifests or
// linked sources before any staged content replaces the installed roots.
func (s installationSnapshot) matches() bool {
	current, err := readInstallationSnapshot(filepath.Base(s.roots.TcDir))
	if err != nil {
		return false
	}
	if s.record == nil {
		return current.record == nil
	}
	if current.record == nil || *current.record != *s.record || !slices.Equal(current.intents, s.intents) {
		return false
	}
	return current.captureMetadata() == nil && current.metadata == s.metadata
}

func snapshotGroupMember(identity string) (*groupMember, error) {
	snapshot, err := readInstallationSnapshot(identity)
	if err != nil {
		return nil, err
	}
	return &groupMember{identity: identity, installed: snapshot, intents: slices.Clone(snapshot.intents)}, nil
}

func componentState(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var state strings.Builder
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return "", err
		}
		state.WriteString(entry.Name())
		state.WriteByte(0)
		state.Write(data)
		state.WriteByte(0)
	}
	return state.String(), nil
}
