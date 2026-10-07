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
	"github.com/Zxilly/cjv/internal/lifecycle"
	"github.com/Zxilly/cjv/internal/progress"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
)

// AutoInstallFunc is the test seam for auto-install. Production defaults to
// internal/lifecycle so main does not wire resolve back up to CLI.
var AutoInstallFunc func(ctx context.Context, input string, targets []string) error

// AutoInstallComponentsFunc is the test seam for missing component installs.
var AutoInstallComponentsFunc func(ctx context.Context, input string, components []string) error

// autoInstallProgress is the adapter auto-install reports to. The proxy path
// runs before any renderer exists and its stdout belongs to the proxied
// tool, so every message and the download bar go to stderr.
func autoInstallProgress() progress.Sink {
	return progress.NewText(os.Stderr, os.Stderr)
}

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
	_, settings, settingsErr := config.LoadDefaultSettings()
	selected, err := toolchain.SelectActive(settings, settingsErr, tcOverride)
	if err != nil {
		return ActiveToolchain{}, err
	}
	if settingsErr != nil {
		slog.Warn("failed to load settings", "error", settingsErr)
	}

	tcName, source, targets, components := selected.Name, selected.Source, selected.Targets, selected.Components
	sink := autoInstallProgress()
	var install func(context.Context) error
	if installFunc := autoInstallFunc(sink); shouldAutoInstall(settings) && installFunc != nil {
		install = func(ctx context.Context) error {
			sink.Report(progress.Event{Kind: progress.AutoInstalling, Subject: tcName})
			if installErr := installFunc(ctx, tcName, targets); installErr != nil {
				sink.Report(progress.Event{Kind: progress.AutoInstallFailed, Subject: tcName, Err: installErr})
				return &cjverr.ToolchainNotInstalledError{Name: tcName}
			}
			return nil
		}
	}
	tcDir, displayName, err := toolchain.PrepareActive(ctx, tcName, install)
	if err != nil {
		return ActiveToolchain{}, err
	}

	if err := ensureTargets(ctx, displayName, tcDir, settings, targets, sink); err != nil {
		return ActiveToolchain{}, err
	}

	if err := ensureComponents(ctx, displayName, tcDir, settings, components, sink); err != nil {
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
	if filepath.IsAbs(host.Name) {
		return ActiveToolchain{}, fmt.Errorf("custom toolchain %s has no published targets", host.Name)
	}

	parsed, err := toolchain.InstalledRelease(host.Dir)
	if err != nil {
		return ActiveToolchain{}, err
	}
	if parsed.IsCustom() || parsed.Channel == toolchain.UnknownChannel || parsed.Version == "" {
		return ActiveToolchain{}, fmt.Errorf("cannot resolve target %q: host toolchain %q has no channel/version", target, host.Name)
	}

	_, settings, settingsErr := config.LoadDefaultSettings()
	if settingsErr != nil {
		slog.Warn("failed to load settings", "error", settingsErr)
	}
	tuple, err := targetPlatformForInstallation(host.Dir, settings, target)
	if err != nil {
		return ActiveToolchain{}, err
	}

	name := targetIdentity(host.Name, parsed, tuple)
	tcDir, err := toolchain.FindInstalled(name)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ActiveToolchain{}, &cjverr.ToolchainNotInstalledError{Name: name.String()}
		}
		return ActiveToolchain{}, err
	}

	if err := targetMatchesRelease(tcDir, parsed); err != nil {
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

func ensureTargets(ctx context.Context, tcInput, tcDir string, settings *config.Settings, targets []string, sink progress.Sink) error {
	if len(targets) == 0 {
		return nil
	}

	host, err := toolchain.InstalledRelease(tcDir)
	if err != nil {
		return err
	}
	if host.IsCustom() || host.Channel == toolchain.UnknownChannel || host.Version == "" {
		return nil
	}

	var missingTargets []string
	var missingNames []string
	for _, target := range targets {
		tuple, err := targetPlatformForInstallation(tcDir, settings, target)
		if err != nil {
			return err
		}
		name := targetIdentity(filepath.Base(tcDir), host, tuple)
		dir, findErr := toolchain.FindInstalled(name)
		if findErr == nil {
			findErr = targetMatchesRelease(dir, host)
		}
		if err := findErr; err != nil {
			if !errors.Is(err, os.ErrNotExist) {
				var missing *cjverr.ToolchainNotInstalledError
				if !errors.As(err, &missing) {
					return err
				}
			}
			missingTargets = append(missingTargets, target)
			missingNames = append(missingNames, name.String())
		}
	}
	if len(missingTargets) == 0 {
		return nil
	}

	installFunc := autoInstallTargetsFunc(sink)
	if !shouldAutoInstall(settings) || installFunc == nil {
		return &cjverr.ToolchainNotInstalledError{Name: missingNames[0]}
	}

	subject := strings.Join(missingNames, ", ")
	sink.Report(progress.Event{Kind: progress.AutoInstalling, Subject: subject})
	if installErr := installFunc(ctx, tcInput, missingTargets); installErr != nil {
		sink.Report(progress.Event{Kind: progress.AutoInstallFailed, Subject: subject, Err: installErr})
		return &cjverr.ToolchainNotInstalledError{Name: missingNames[0]}
	}

	for _, missingName := range missingNames {
		parsed, err := toolchain.ParseToolchainName(missingName)
		if err != nil {
			return err
		}
		dir, err := toolchain.FindInstalled(parsed)
		if err != nil {
			return &cjverr.ToolchainNotInstalledError{Name: missingName}
		}
		if err := targetMatchesRelease(dir, host); err != nil {
			return err
		}
	}
	return nil
}

func shouldAutoInstall(settings *config.Settings) bool {
	return settings != nil && settings.AutoInstall
}

func autoInstallFunc(sink progress.Sink) func(context.Context, string, []string) error {
	if AutoInstallFunc != nil {
		return AutoInstallFunc
	}
	return func(ctx context.Context, input string, targets []string) error {
		return lifecycle.Install(ctx, lifecycle.InstallRequest{Toolchain: input, Targets: targets}, lifecycle.Options{Progress: sink})
	}
}

func autoInstallTargetsFunc(sink progress.Sink) func(context.Context, string, []string) error {
	if AutoInstallFunc != nil {
		return AutoInstallFunc
	}
	return func(ctx context.Context, input string, targets []string) error {
		return lifecycle.InstallTargetsForToolchain(ctx, input, targets, lifecycle.Options{Progress: sink})
	}
}

func autoInstallComponentsFunc(sink progress.Sink) func(context.Context, string, []string) error {
	if AutoInstallComponentsFunc != nil {
		return AutoInstallComponentsFunc
	}
	return func(ctx context.Context, input string, components []string) error {
		return lifecycle.InstallComponentsForToolchain(ctx, input, components, lifecycle.Options{Progress: sink})
	}
}

func ensureComponents(ctx context.Context, tcInput, tcDir string, settings *config.Settings, components []string, sink progress.Sink) error {
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

	installComponentsFunc := autoInstallComponentsFunc(sink)
	if !shouldAutoInstall(settings) || installComponentsFunc == nil {
		return &cjverr.ComponentNotInstalledError{
			Toolchain: filepath.Base(tcDir),
			Component: asStrings[0],
		}
	}

	subject := strings.Join(asStrings, ", ")
	sink.Report(progress.Event{Kind: progress.AutoInstalling, Subject: subject})
	if err := installComponentsFunc(ctx, tcInput, asStrings); err != nil {
		sink.Report(progress.Event{Kind: progress.AutoInstallFailed, Subject: subject, Err: err})
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
	return sdktarget.CurrentTargetTuple(defaultHost, target)
}

func targetPlatformForInstallation(dir string, settings *config.Settings, environment string) (string, error) {
	record, err := toolchain.ReadInstallation(dir)
	if err != nil {
		return "", err
	}
	if record.Tuple == "" {
		return targetPlatformKey(settings, environment)
	}
	id, err := sdktarget.ParseIdentity(record.Tuple)
	if err != nil {
		return "", err
	}
	cross, err := id.WithEnvironment(environment)
	return cross.Tuple(), err
}

func targetIdentity(hostIdentity string, release toolchain.ToolchainName, tuple string) toolchain.ToolchainName {
	identity, _ := toolchain.ParseToolchainName(hostIdentity)
	if identity.Version == "" {
		release.Version = ""
	}
	release.Target = tuple
	release.Host = ""
	return release
}

func targetMatchesRelease(dir string, host toolchain.ToolchainName) error {
	target, err := toolchain.InstalledRelease(dir)
	if err != nil {
		return err
	}
	if target.Channel != host.Channel || target.Version != host.Version {
		return &cjverr.ToolchainNotInstalledError{Name: filepath.Base(dir)}
	}
	return nil
}
