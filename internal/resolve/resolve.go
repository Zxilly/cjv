package resolve

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// AutoInstallFunc is wired by main to avoid importing cli from lower-level packages.
var AutoInstallFunc func(ctx context.Context, input string, targets []string) error

// AutoInstallComponentsFunc auto-installs missing components on demand. Wired
// by main; must be safe to leave nil in tests where component installation
// is not exercised.
var AutoInstallComponentsFunc func(ctx context.Context, input string, components []string) error

type ActiveToolchain struct {
	Dir        string
	Name       string
	Source     config.OverrideSource
	Targets    []string
	Components []string
}

func Active(ctx context.Context, tcOverride string) (ActiveToolchain, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	toolchain.CleanupStagingDirs()

	settings, settingsErr := loadSettings()
	tcName, source, targets, components, err := resolveName(settings, settingsErr, tcOverride)
	if err != nil {
		return ActiveToolchain{}, err
	}
	if settingsErr != nil {
		slog.Warn("failed to load settings", "error", settingsErr)
	}

	// Share the parse → reject-target → find-installed core with
	// toolchain.ResolveActiveToolchain; the auto-install retry and the
	// target/component ensuring below are the deliberate extra behavior of the
	// proxy path.
	tcDir, displayName, parsed, err := toolchain.FindActiveDir(tcName)
	if err != nil {
		var notInstalled *cjverr.ToolchainNotInstalledError
		if errors.As(err, &notInstalled) && !parsed.IsCustom() && shouldAutoInstall(settings) && AutoInstallFunc != nil {
			fmt.Fprintln(os.Stderr, i18n.T("AutoInstalling", i18n.MsgData{"Name": tcName}))
			if installErr := AutoInstallFunc(ctx, tcName, targets); installErr != nil {
				fmt.Fprintf(os.Stderr, "%s\n", i18n.T("AutoInstallFailed", i18n.MsgData{
					"Name": tcName,
					"Err":  installErr.Error(),
				}))
				return ActiveToolchain{}, &cjverr.ToolchainNotInstalledError{Name: tcName}
			}
			tcDir, displayName, _, err = toolchain.FindActiveDir(tcName)
		}
		if err != nil {
			return ActiveToolchain{}, err
		}
	}

	if err := ensureTargets(ctx, displayName, tcDir, settings, targets); err != nil {
		return ActiveToolchain{}, err
	}

	if err := ensureComponents(ctx, displayName, tcDir, settings, components); err != nil {
		return ActiveToolchain{}, err
	}

	return ActiveToolchain{
		Dir:        tcDir,
		Name:       displayName,
		Source:     source,
		Targets:    targets,
		Components: components,
	}, nil
}

func loadSettings() (*config.Settings, error) {
	sf, err := config.DefaultSettingsFile()
	if err != nil {
		return nil, err
	}
	return sf.Load()
}

// ActiveTarget resolves the installed cross-compilation target SDK for the
// given target suffix, layered on the host toolchain selected by tcOverride.
// Unlike Active (which rejects target variants as the active toolchain), this
// returns an ActiveToolchain whose Dir is the target SDK's own directory so
// callers can derive a standalone cross-compile environment from it, exactly
// as the host toolchain is derived. Name remains the host toolchain identity
// (e.g. "lts-1.0.5") — the logical toolchain being used — while Dir/root is the
// target SDK and Targets names the cross target. The target SDK must already be
// installed (cjv install <toolchain> --target <suffix>); this does not
// auto-install it.
func ActiveTarget(ctx context.Context, tcOverride, target string) (ActiveToolchain, error) {
	host, err := Active(ctx, tcOverride)
	if err != nil {
		return ActiveToolchain{}, err
	}

	parsed, err := toolchain.ParseToolchainName(host.Name)
	if err != nil {
		return ActiveToolchain{}, err
	}
	if parsed.IsCustom() || parsed.Channel == toolchain.UnknownChannel || parsed.Version == "" {
		return ActiveToolchain{}, fmt.Errorf("cannot resolve target %q: host toolchain %q has no channel/version", target, host.Name)
	}

	settings, settingsErr := loadSettings()
	if settingsErr != nil {
		slog.Warn("failed to load settings", "error", settingsErr)
	}
	tuple, err := targetPlatformKey(settings, target)
	if err != nil {
		return ActiveToolchain{}, err
	}

	name := toolchain.ToolchainName{
		Channel: parsed.Channel,
		Version: parsed.Version,
		Target:  tuple,
	}
	tcDir, err := toolchain.FindInstalled(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ActiveToolchain{}, &cjverr.ToolchainNotInstalledError{Name: name.String()}
		}
		return ActiveToolchain{}, err
	}

	return ActiveToolchain{
		Dir:    tcDir,
		Name:   host.Name,
		Source: host.Source,
		// Components are not carried over: they were only ensured on the host,
		// not verified against the target SDK dir, so claiming them here would
		// mislabel the target SDK's component set.
		Targets:    []string{target},
		Components: nil,
	}, nil
}

func resolveName(settings *config.Settings, settingsErr error, tcOverride string) (string, config.OverrideSource, []string, []string, error) {
	if tcOverride != "" {
		return tcOverride, config.SourceUnknown, nil, nil, nil
	}
	if envTC := os.Getenv(config.EnvToolchain); envTC != "" {
		return envTC, config.SourceEnv, nil, nil, nil
	}
	if settingsErr != nil {
		return "", config.SourceUnknown, nil, nil, settingsErr
	}
	cwd, err := os.Getwd()
	if err != nil {
		return "", config.SourceUnknown, nil, nil, fmt.Errorf("failed to get working directory: %w", err)
	}
	resolved, err := config.ResolveToolchainConfig(settings, cwd)
	if err != nil {
		return "", config.SourceUnknown, nil, nil, err
	}
	return resolved.Name, resolved.Source, resolved.Targets, resolved.Components, nil
}

func ensureTargets(ctx context.Context, tcInput, tcDir string, settings *config.Settings, targets []string) error {
	if len(targets) == 0 {
		return nil
	}

	host, err := toolchain.ParseToolchainName(filepath.Base(tcDir))
	if err != nil {
		return err
	}
	if host.IsCustom() || host.Channel == toolchain.UnknownChannel || host.Version == "" {
		return nil
	}

	var missingTargets []string
	var missingNames []string
	for _, target := range targets {
		tuple, err := targetPlatformKey(settings, target)
		if err != nil {
			return err
		}
		name := toolchain.ToolchainName{
			Channel: host.Channel,
			Version: host.Version,
			Target:  tuple,
		}
		if _, err := toolchain.FindInstalled(name); err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				return err
			}
			missingTargets = append(missingTargets, target)
			missingNames = append(missingNames, name.String())
		}
	}
	if len(missingTargets) == 0 {
		return nil
	}

	if !shouldAutoInstall(settings) || AutoInstallFunc == nil {
		return &cjverr.ToolchainNotInstalledError{Name: missingNames[0]}
	}

	fmt.Fprintln(os.Stderr, i18n.T("AutoInstalling", i18n.MsgData{"Name": strings.Join(missingNames, ", ")}))
	if installErr := AutoInstallFunc(ctx, tcInput, missingTargets); installErr != nil {
		fmt.Fprintf(os.Stderr, "%s\n", i18n.T("AutoInstallFailed", i18n.MsgData{
			"Name": strings.Join(missingNames, ", "),
			"Err":  installErr.Error(),
		}))
		return &cjverr.ToolchainNotInstalledError{Name: missingNames[0]}
	}

	for _, missingName := range missingNames {
		parsed, err := toolchain.ParseToolchainName(missingName)
		if err != nil {
			return err
		}
		if _, err := toolchain.FindInstalled(parsed); err != nil {
			return &cjverr.ToolchainNotInstalledError{Name: missingName}
		}
	}
	return nil
}

func shouldAutoInstall(settings *config.Settings) bool {
	return settings != nil && settings.AutoInstall
}

func ensureComponents(ctx context.Context, tcInput, tcDir string, settings *config.Settings, components []string) error {
	if len(components) == 0 {
		return nil
	}

	parsedNames, err := component.NormalizeList(components)
	if err != nil {
		return err
	}

	var missingNames []component.Name
	for _, n := range parsedNames {
		if !component.IsInstalled(tcDir, n) {
			missingNames = append(missingNames, n)
		}
	}
	if len(missingNames) == 0 {
		return nil
	}

	asStrings := make([]string, len(missingNames))
	for i, n := range missingNames {
		asStrings[i] = string(n)
	}

	if !shouldAutoInstall(settings) || AutoInstallComponentsFunc == nil {
		return &cjverr.ComponentNotInstalledError{
			Toolchain: filepath.Base(tcDir),
			Component: asStrings[0],
		}
	}

	fmt.Fprintln(os.Stderr, i18n.T("AutoInstalling", i18n.MsgData{
		"Name": strings.Join(asStrings, ", "),
	}))
	if err := AutoInstallComponentsFunc(ctx, tcInput, asStrings); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", i18n.T("AutoInstallFailed", i18n.MsgData{
			"Name": strings.Join(asStrings, ", "),
			"Err":  err.Error(),
		}))
		return &cjverr.ComponentNotInstalledError{
			Toolchain: filepath.Base(tcDir),
			Component: asStrings[0],
		}
	}
	for _, n := range missingNames {
		if !component.IsInstalled(tcDir, n) {
			return &cjverr.ComponentNotInstalledError{
				Toolchain: filepath.Base(tcDir),
				Component: string(n),
			}
		}
	}
	return nil
}

func targetPlatformKey(settings *config.Settings, target string) (string, error) {
	defaultHost := ""
	if settings != nil {
		defaultHost = settings.DefaultHost
	}
	return dist.CurrentTargetTuple(defaultHost, target)
}
