package toolchain

import (
	"context"
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
	if filepath.IsAbs(rawName) {
		info, err := os.Stat(rawName)
		if err != nil {
			return "", rawName, ToolchainName{Custom: rawName}, err
		}
		if !info.IsDir() {
			return "", rawName, ToolchainName{}, fmt.Errorf("toolchain path %s is not a directory", rawName)
		}
		return filepath.Clean(rawName), rawName, ToolchainName{Custom: rawName}, nil
	}
	parsed, err = ParseActiveName(rawName)
	if err != nil {
		return "", rawName, parsed, err
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

// ParseActiveName validates a named host toolchain, including names stored in
// defaults and directory overrides. Target SDKs cannot become active hosts.
func ParseActiveName(rawName string) (ToolchainName, error) {
	parsed, err := ParseToolchainName(rawName)
	if err != nil {
		return parsed, err
	}
	if parsed.Target != "" {
		hostName := ToolchainName{Channel: parsed.Channel, Version: parsed.Version}.String()
		return parsed, fmt.Errorf("target variant %q cannot be used as the active toolchain; use host toolchain %q and configure targets instead", rawName, hostName)
	}
	return parsed, nil
}

// PrepareActive recovers installed state and locates a valid active host. A
// non-nil install permits installation only for a genuinely missing official
// toolchain; invalid selections and filesystem failures are never retried as
// installs. Installation must leave a valid host, verified through the same
// lookup as the initial attempt. Callers own installation policy and progress.
func PrepareActive(ctx context.Context, rawName string, install func(context.Context) error) (dir, displayName string, err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := RecoverHomeContext(ctx); err != nil {
		return "", rawName, err
	}
	dir, displayName, parsed, err := FindActiveDir(rawName)
	var missing *cjverr.ToolchainNotInstalledError
	if install == nil || !errors.As(err, &missing) || parsed.IsCustom() {
		return dir, displayName, err
	}
	if err := install(ctx); err != nil {
		return "", rawName, err
	}
	dir, displayName, _, err = FindActiveDir(rawName)
	return dir, displayName, err
}

// SelectActive applies the common selector precedence while retaining project
// targets and components. An explicit selector or environment selection can
// still be inspected when unrelated settings are unreadable.
func SelectActive(settings *config.Settings, settingsErr error, override string) (config.ToolchainConfig, error) {
	if override != "" {
		return config.ToolchainConfig{Name: override}, nil
	}
	if name := os.Getenv(config.EnvToolchain); name != "" {
		return config.ToolchainConfig{Name: name, Source: config.SourceEnv}, nil
	}
	if settingsErr != nil {
		return config.ToolchainConfig{}, settingsErr
	}
	cwd, err := os.Getwd()
	if err != nil {
		return config.ToolchainConfig{}, fmt.Errorf("failed to get working directory: %w", err)
	}
	return config.ResolveToolchainConfig(settings, cwd)
}

// ResolveActiveToolchain resolves the current active toolchain directory, name,
// and source WITHOUT auto-installing (used by status/management commands). On
// error, tcName may still contain the configured (but uninstalled) toolchain
// name. resolve.Active is the auto-installing counterpart for the proxy path.
func ResolveActiveToolchain() (tcDir string, tcName string, source config.OverrideSource, err error) {
	return InspectActive(context.Background(), "")
}

// InspectActive resolves selection and provenance without installing content.
func InspectActive(ctx context.Context, override string) (tcDir string, tcName string, source config.OverrideSource, err error) {
	_, settings, settingsErr := config.LoadDefaultSettings()
	selected, err := SelectActive(settings, settingsErr, override)
	if err != nil {
		return "", "", 0, err
	}
	rawName, source := selected.Name, selected.Source

	dir, displayName, err := PrepareActive(ctx, rawName, nil)
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
