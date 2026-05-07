package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	clisettings "github.com/Zxilly/cjv/internal/cli/settings"
	componentlib "github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/dist"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var (
	componentToolchain         string
	componentAddForce          bool
	componentListInstalledOnly bool
	componentListQuiet         bool
)

var componentCmd = &cobra.Command{
	Use:   "component",
	Short: i18n.T("ComponentSubcmdShort", nil),
	Args:  cobra.NoArgs,
}

var componentAddCmd = &cobra.Command{
	Use:   "add <name>...",
	Short: i18n.T("ComponentAddShort", nil),
	Args:  cobra.MinimumNArgs(1),
	RunE:  runComponentAdd,
}

var componentRemoveCmd = &cobra.Command{
	Use:     "remove <name>...",
	Aliases: []string{"uninstall", "rm", "delete", "del"},
	Short:   i18n.T("ComponentRemoveShort", nil),
	Args:    cobra.MinimumNArgs(1),
	RunE:    runComponentRemove,
}

var componentListCmd = &cobra.Command{
	Use:   "list",
	Short: i18n.T("ComponentListShort", nil),
	Args:  cobra.NoArgs,
	RunE:  runComponentList,
}

func init() {
	componentCmd.PersistentFlags().StringVar(&componentToolchain, "toolchain", "", i18n.T("ComponentFlagToolchain", nil))
	componentAddCmd.Flags().BoolVar(&componentAddForce, "force", false, i18n.T("InstallFlagForce", nil))
	componentListCmd.Flags().BoolVar(&componentListInstalledOnly, "installed", false, i18n.T("ComponentFlagInstalled", nil))
	componentListCmd.Flags().BoolVarP(&componentListQuiet, "quiet", "q", false, i18n.T("ComponentFlagQuiet", nil))

	componentCmd.AddCommand(componentAddCmd)
	componentCmd.AddCommand(componentRemoveCmd)
	componentCmd.AddCommand(componentListCmd)
	rootCmd.AddCommand(componentCmd)
}

// resolveToolchainArg falls back to the active toolchain when flagValue is empty.
func resolveToolchainArg(flagValue string) (string, toolchain.ToolchainName, error) {
	if flagValue != "" {
		parsed, err := toolchain.ParseToolchainName(flagValue)
		if err != nil {
			return "", toolchain.ToolchainName{}, err
		}
		dir, err := toolchain.FindInstalled(parsed)
		if err != nil {
			return "", toolchain.ToolchainName{}, err
		}
		actual, err := toolchain.ParseToolchainName(filepath.Base(dir))
		if err != nil {
			return "", toolchain.ToolchainName{}, err
		}
		return dir, actual, nil
	}
	dir, name, _, err := toolchain.ResolveActiveToolchain()
	if err != nil {
		return "", toolchain.ToolchainName{}, err
	}
	parsed, err := toolchain.ParseToolchainName(name)
	if err != nil {
		return "", toolchain.ToolchainName{}, err
	}
	return dir, parsed, nil
}

func runComponentAdd(cmd *cobra.Command, args []string) error {
	tcDir, tcName, err := resolveToolchainArg(componentToolchain)
	if err != nil {
		return err
	}
	if tcName.IsCustom() {
		return &cjverr.ComponentRequiresHostError{Component: args[0]}
	}
	return installComponentsList(cmd.Context(), filepath.Base(tcDir), args, componentAddForce, false)
}

func runComponentRemove(cmd *cobra.Command, args []string) error {
	parsed, parseErrs := parseComponentRemoveArgs(args)
	if len(parsed) == 0 {
		return errors.Join(parseErrs...)
	}

	tcDir, _, err := resolveToolchainArg(componentToolchain)
	if err != nil {
		return err
	}

	roots, err := componentlib.RootsFor(filepath.Base(tcDir))
	if err != nil {
		return err
	}

	var removeErrs []error
	removeErrs = append(removeErrs, parseErrs...)
	for _, c := range parsed {
		if !componentlib.IsInstalled(tcDir, c) {
			removeErrs = append(removeErrs, &cjverr.ComponentNotInstalledError{
				Toolchain: filepath.Base(tcDir),
				Component: string(c),
			})
			continue
		}
		fmt.Println(i18n.T("RemovingComponent", i18n.MsgData{"Component": string(c)}))
		if err := componentlib.Remove(roots, c); err != nil {
			removeErrs = append(removeErrs, err)
			continue
		}
		color.Green(i18n.T("ComponentRemoved", i18n.MsgData{
			"Toolchain": filepath.Base(tcDir),
			"Component": string(c),
		}))
	}
	return errors.Join(removeErrs...)
}

func runComponentList(cmd *cobra.Command, args []string) error {
	tcDir, tcName, err := resolveToolchainArg(componentToolchain)
	if err != nil {
		return err
	}

	installed, err := componentlib.ListInstalled(tcDir)
	if err != nil {
		return err
	}

	available := componentlib.KnownComponents()
	if !tcName.IsCustom() {
		platformKey := tcName.PlatformKey
		if platformKey == "" {
			_, settings, err := clisettings.LoadSettings()
			if err != nil {
				return err
			}
			platformKey, err = dist.CurrentPlatformKey(settings.DefaultHost)
			if err != nil {
				return err
			}
		}
		available = componentlib.AvailableComponents(tcName, platformKey)
	} else {
		available = nil
	}

	if componentListQuiet {
		for _, n := range installed {
			fmt.Println(string(n))
		}
		if !componentListInstalledOnly {
			for _, n := range available {
				if slices.Contains(installed, n) {
					continue
				}
				fmt.Println(string(n))
			}
		}
		return nil
	}

	if len(installed) == 0 && componentListInstalledOnly {
		fmt.Println(i18n.T("NoComponentsInstalled", nil))
		return nil
	}

	for _, n := range installed {
		fmt.Printf("%s (%s)\n", string(n), i18n.T("StatusInstalled", nil))
	}
	if !componentListInstalledOnly {
		for _, n := range available {
			if slices.Contains(installed, n) {
				continue
			}
			fmt.Printf("%s (%s)\n", string(n), i18n.T("AvailableComponents", nil))
		}
	}
	return nil
}

func parseComponentRemoveArgs(args []string) ([]componentlib.Name, []error) {
	var out []componentlib.Name
	var errs []error
	seen := make(map[componentlib.Name]bool)
	seenUnknown := make(map[string]bool)
	for _, raw := range args {
		for part := range strings.SplitSeq(raw, ",") {
			part = strings.TrimSpace(part)
			if part == "" {
				continue
			}
			n, err := componentlib.ParseName(part)
			if err != nil {
				if !seenUnknown[part] {
					seenUnknown[part] = true
					errs = append(errs, err)
				}
				continue
			}
			if !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	return out, errs
}
