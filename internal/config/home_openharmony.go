//go:build openharmony

package config

import (
	"errors"
	"path/filepath"
)

// HDC and application shells can expose different homes. Never turn a missing
// user environment into an installation under the system root on OpenHarmony.
func validateUserHome(home string) error {
	if !filepath.IsAbs(home) || filepath.Dir(filepath.Clean(home)) == filepath.Clean(home) {
		return errors.New("OpenHarmony requires an absolute user HOME distinct from the filesystem root; run cjv in the user's terminal environment")
	}
	return nil
}
