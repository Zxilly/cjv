package lifecycle

import (
	"errors"
	"github.com/Zxilly/cjv/internal/component"
	"os"
	"path/filepath"
)

func componentState(dir string) (string, error) {
	var state string
	entries, err := os.ReadDir(dir)
	if errors.Is(err, os.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return "", err
		}
		state += entry.Name() + "\x00" + string(data) + "\x00"
	}
	return state, nil
}

func installationComponentState(dir string) (string, error) {
	state, err := componentState(filepath.Join(dir, component.MetaDir))
	if err != nil {
		return "", err
	}
	roots, err := component.RootsFor(filepath.Base(dir))
	if err != nil {
		return "", err
	}
	intents, err := component.InstalledIntents(roots)
	if err != nil {
		return "", err
	}
	for _, intent := range intents {
		state += string(intent.Name) + "\x00" + intent.Source + "\x00"
	}
	return state, nil
}
