package cli

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/Zxilly/cjv/internal/cli/output"
	componentlib "github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/env"
	"github.com/Zxilly/cjv/internal/resolve"
	"github.com/spf13/cobra"
)

const jsonFlagName = "json"

func parseJSONModeFlag(arg string) (bool, bool, error) {
	if arg == "--"+jsonFlagName {
		return true, true, nil
	}
	prefix := "--" + jsonFlagName + "="
	if !strings.HasPrefix(arg, prefix) {
		return false, false, nil
	}
	value, err := strconv.ParseBool(strings.TrimPrefix(arg, prefix))
	if err != nil {
		return true, false, fmt.Errorf("invalid --%s value %q", jsonFlagName, strings.TrimPrefix(arg, prefix))
	}
	return true, value, nil
}

func applyJSONModeFlag(arg string) (bool, error) {
	matched, value, err := parseJSONModeFlag(arg)
	if err != nil || !matched {
		return matched, err
	}
	output.SetJSONMode(value)
	return true, nil
}

func stripJSONModeFlagPrefix(args []string, allowAfterPlusToolchain bool) ([]string, error) {
	out := make([]string, 0, len(args))
	scanning := true
	sawPlusToolchain := false
	for i, arg := range args {
		if scanning {
			if arg == "--" {
				out = append(out, args[i+1:]...)
				return out, nil
			}
			matched, err := applyJSONModeFlag(arg)
			if err != nil {
				return nil, err
			}
			if matched {
				continue
			}
			if allowAfterPlusToolchain && !sawPlusToolchain && strings.HasPrefix(arg, "+") && len(arg) > 1 {
				sawPlusToolchain = true
				out = append(out, arg)
				continue
			}
			scanning = false
		}
		out = append(out, arg)
	}
	return out, nil
}

func newEnvsetupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "envsetup [+toolchain] [--target=SUFFIX] [--shell=TYPE]",
		Short: "Print shell commands to configure Cangjie runtime environment",
		Long: `Output shell commands that set environment variables for the active Cangjie toolchain.

Usage:
  eval "$(cjv envsetup)"          # bash/zsh
  cjv envsetup | source           # fish
  cjv envsetup | Invoke-Expression  # powershell`,
		Args: cobra.ArbitraryArgs,
		RunE: envsetupRun,
	}
	// Unlike exec/run, envsetup does not forward arguments to a child process,
	// so it lets cobra parse flags normally: --json is the global persistent
	// flag and +toolchain is an ordinary positional argument.
	cmd.Flags().String("shell", "", "shell type to format for (bash, fish, powershell, cmd)")
	cmd.Flags().String("target", "", "cross-compilation target suffix (e.g. ohos) to emit the environment for the installed target SDK")
	return cmd
}

func envsetupRun(cmd *cobra.Command, args []string) error {
	shellFlag, _ := cmd.Flags().GetString("shell")
	targetFlag, _ := cmd.Flags().GetString("target")
	if output.IsJSON() {
		return envsetupRunJSON(cmd, args, targetFlag)
	}
	return envsetupRunWithShell(cmd, args, shellFlag, targetFlag)
}

type envsetupData struct {
	active resolve.ActiveToolchain
	cfg    *env.EnvConfig
	home   string
	bin    string
	source string
}

type envsetupJSONResult struct {
	SchemaVersion int                     `json:"schema_version"`
	Toolchain     envsetupToolchainJSON   `json:"toolchain"`
	CJV           envsetupCJVJSON         `json:"cjv"`
	Env           envsetupEnvironmentJSON `json:"env"`
}

type envsetupToolchainJSON struct {
	Name       string   `json:"name"`
	Root       string   `json:"root"`
	Source     string   `json:"source"`
	Targets    []string `json:"targets"`
	Components []string `json:"components"`
}

type envsetupCJVJSON struct {
	Home string `json:"home"`
	Bin  string `json:"bin"`
}

type envsetupEnvironmentJSON struct {
	Vars        map[string]string       `json:"vars"`
	Path        envsetupPathJSON        `json:"path"`
	LibraryPath envsetupLibraryPathJSON `json:"library_path"`
}

type envsetupPathJSON struct {
	Prepend []string `json:"prepend"`
	Append  []string `json:"append"`
}

type envsetupLibraryPathJSON struct {
	Key     *string  `json:"key"`
	Prepend []string `json:"prepend"`
}

func (r envsetupJSONResult) Text() string { return "" }

func loadEnvsetupData(ctx context.Context, tcOverride, target string) (envsetupData, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	var active resolve.ActiveToolchain
	var err error
	if target != "" {
		active, err = resolve.ActiveTarget(ctx, tcOverride, target)
	} else {
		active, err = resolve.Active(ctx, tcOverride)
	}
	if err != nil {
		return envsetupData{}, err
	}
	cfg := env.LoadToolchainEnv(active.Dir, componentlib.ApplyEnv)

	home, err := config.Home()
	if err != nil {
		return envsetupData{}, err
	}
	bin, err := config.BinDir()
	if err != nil {
		return envsetupData{}, err
	}

	return envsetupData{
		active: active,
		cfg:    cfg,
		home:   home,
		bin:    bin,
		source: envsetupSource(tcOverride, active.Source),
	}, nil
}

func envsetupSource(tcOverride string, source config.OverrideSource) string {
	if tcOverride != "" {
		return "argument"
	}
	switch source {
	case config.SourceEnv:
		return "env"
	case config.SourceOverride:
		return "override"
	case config.SourceToolchainFile:
		return "toolchain-file"
	case config.SourceDefault:
		return "default"
	default:
		return "unknown"
	}
}

func envsetupVarsForJSON(cfg *env.EnvConfig) map[string]string {
	vars := make(map[string]string)
	if cfg == nil {
		return vars
	}
	libraryKey := env.RuntimeLibraryPathKey()
	for k, v := range cfg.Vars {
		if k == "" || k == config.EnvToolchain || k == config.EnvRecursionCount || k == libraryKey {
			continue
		}
		vars[k] = v
	}
	return vars
}

// jsonStrings normalizes a slice for JSON output so an empty or nil input
// serializes as [] rather than null, keeping the schema's array fields a
// stable shape for typed consumers.
func jsonStrings(s []string) []string {
	return append(make([]string, 0, len(s)), s...)
}

func envsetupResultFromData(data envsetupData) envsetupJSONResult {
	libraryKey := env.RuntimeLibraryPathKey()
	var libraryKeyPtr *string
	if libraryKey != "" {
		libraryKeyPtr = &libraryKey
	}

	return envsetupJSONResult{
		SchemaVersion: 1,
		Toolchain: envsetupToolchainJSON{
			Name:       data.active.Name,
			Root:       data.active.Dir,
			Source:     data.source,
			Targets:    jsonStrings(data.active.Targets),
			Components: jsonStrings(data.active.Components),
		},
		CJV: envsetupCJVJSON{
			Home: data.home,
			Bin:  data.bin,
		},
		Env: envsetupEnvironmentJSON{
			Vars: envsetupVarsForJSON(data.cfg),
			Path: envsetupPathJSON{
				Prepend: jsonStrings(data.cfg.PathPrepend),
				Append:  jsonStrings(data.cfg.PathAppend),
			},
			LibraryPath: envsetupLibraryPathJSON{
				Key:     libraryKeyPtr,
				Prepend: jsonStrings(env.ExistingLibraryPathEntries(data.cfg)),
			},
		},
	}
}

func envsetupRunJSON(cmd *cobra.Command, args []string, target string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	tcOverride, _ := extractPlusToolchainFromArgs(args)

	data, err := loadEnvsetupData(ctx, tcOverride, target)
	if err != nil {
		return err
	}
	return output.RenderTo(cmdOutput(cmd), envsetupResultFromData(data))
}

func envsetupRunWithShell(cmd *cobra.Command, args []string, shellFlag, target string) error {
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}

	tcOverride, _ := extractPlusToolchainFromArgs(args)

	var shellType env.ShellType
	if shellFlag != "" {
		st, err := env.ParseShellFlag(shellFlag)
		if err != nil {
			return err
		}
		shellType = st
	} else {
		st, detected := env.DetectShell()
		if !detected {
			fmt.Fprintln(os.Stderr, "cjv: could not detect shell type, defaulting to posix. Use --shell=TYPE to override (bash, fish, powershell, cmd)")
		}
		shellType = st
	}

	data, err := loadEnvsetupData(ctx, tcOverride, target)
	if err != nil {
		return err
	}

	baseEnv := os.Environ()
	runtimeEnv := env.BuildToolchainEnv(baseEnv, data.cfg)
	diff := env.ComputeEnvDiff(baseEnv, runtimeEnv)
	if len(diff) == 0 {
		return nil
	}

	output := env.FormatEnvDiff(diff, shellType)
	_, _ = fmt.Fprint(cmdOutput(cmd), output)
	return nil
}

func init() {
	rootCmd.AddCommand(newEnvsetupCmd())
}
