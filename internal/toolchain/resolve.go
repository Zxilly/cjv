package toolchain

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/config"
)

// SplitPlusSelector splits an optional leading "+toolchain" selector from args.
// It returns the toolchain name (without the "+"), the remaining args, and
// whether a "+"-prefixed token was present. A bare "+" yields name=="" with
// present==true so a strict caller (the proxy shim) can reject it while a
// lenient caller (cjv exec / envsetup) can ignore it. The single
// implementation keeps the three call sites from drifting on the syntax.
func SplitPlusSelector(args []string) (name string, rest []string, present bool) {
	if len(args) > 0 && strings.HasPrefix(args[0], "+") {
		return args[0][1:], args[1:], true
	}
	return "", args, false
}

// FindActiveDir parses rawName, rejects target variants (which cannot be the
// active toolchain), and locates the installed directory. It performs no
// side effects (no auto-install). It is the shared core behind the read-only
// ResolveActiveToolchain here and resolve.Active, which layers auto-install and
// target/component ensuring on top — keeping the resolution sequence in one
// place so the two callers cannot drift.
//
// On any error the returned displayName is rawName, so callers can still report
// the configured-but-unusable toolchain. parsed is the parsed name (zero on a
// parse error) so callers can branch on e.g. IsCustom.
func FindActiveDir(rawName string) (dir, displayName string, parsed ToolchainName, err error) {
	parsed, err = ParseToolchainName(rawName)
	if err != nil {
		return "", rawName, ToolchainName{}, err
	}
	if parsed.Target != "" {
		hostName := ToolchainName{Channel: parsed.Channel, Version: parsed.Version}.String()
		return "", rawName, parsed, fmt.Errorf("target variant %q cannot be used as the active toolchain; use host toolchain %q and configure targets instead", rawName, hostName)
	}

	found, findErr := FindInstalled(parsed)
	if findErr != nil {
		if !errors.Is(findErr, os.ErrNotExist) {
			return "", rawName, parsed, findErr
		}
		return "", rawName, parsed, &cjverr.ToolchainNotInstalledError{Name: rawName}
	}
	// Use the actual directory name as the display name to avoid showing
	// "unknown-X.Y.Z" for bare version inputs.
	return found, filepath.Base(found), parsed, nil
}

// ResolveActiveToolchain resolves the current active toolchain directory, name,
// and source WITHOUT auto-installing (used by status/management commands). On
// error, tcName may still contain the configured (but uninstalled) toolchain
// name. resolve.Active is the auto-installing counterpart for the proxy path.
func ResolveActiveToolchain() (tcDir string, tcName string, source config.OverrideSource, err error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", "", 0, fmt.Errorf("failed to get working directory: %w", err)
	}
	sf, err := config.DefaultSettingsFile()
	if err != nil {
		return "", "", 0, err
	}
	settings, err := sf.Load()
	if err != nil {
		return "", "", 0, err
	}

	rawName, source, err := config.ResolveToolchain(settings, cwd)
	if err != nil {
		return "", "", 0, err
	}

	dir, displayName, _, err := FindActiveDir(rawName)
	if err != nil {
		var notInstalled *cjverr.ToolchainNotInstalledError
		if !errors.As(err, &notInstalled) {
			// Preserve provenance so the user knows which config source supplied
			// the unusable name. ToolchainNotInstalledError already carries the
			// name and is handled specially by callers (e.g. show), so pass it
			// through unchanged.
			return "", displayName, source, fmt.Errorf("toolchain %q (from %s): %w", rawName, source, err)
		}
		return "", displayName, source, err
	}
	return dir, displayName, source, nil
}
