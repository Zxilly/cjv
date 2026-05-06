package cli

import (
	"fmt"
	"path/filepath"
	"slices"

	"github.com/Zxilly/cjv/internal/cjverr"
	componentlib "github.com/Zxilly/cjv/internal/component"
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
	parsed, err := componentlib.NormalizeList(args)
	if err != nil {
		return err
	}

	tcDir, _, err := resolveToolchainArg(componentToolchain)
	if err != nil {
		return err
	}

	roots, err := componentlib.RootsFor(filepath.Base(tcDir))
	if err != nil {
		return err
	}

	for _, c := range parsed {
		if !componentlib.IsInstalled(tcDir, c) {
			return &cjverr.ComponentNotInstalledError{
				Toolchain: filepath.Base(tcDir),
				Component: string(c),
			}
		}
		fmt.Println(i18n.T("RemovingComponent", i18n.MsgData{"Component": string(c)}))
		if err := componentlib.Remove(roots, c); err != nil {
			return err
		}
		color.Green(i18n.T("ComponentRemoved", i18n.MsgData{
			"Toolchain": filepath.Base(tcDir),
			"Component": string(c),
		}))
	}
	return nil
}

func runComponentList(cmd *cobra.Command, args []string) error {
	tcDir, _, err := resolveToolchainArg(componentToolchain)
	if err != nil {
		return err
	}

	installed, err := componentlib.ListInstalled(tcDir)
	if err != nil {
		return err
	}

	if componentListQuiet {
		for _, n := range installed {
			fmt.Println(string(n))
		}
		if !componentListInstalledOnly {
			for _, n := range componentlib.KnownComponents() {
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
		for _, n := range componentlib.KnownComponents() {
			if slices.Contains(installed, n) {
				continue
			}
			fmt.Printf("%s (%s)\n", string(n), i18n.T("AvailableComponents", nil))
		}
	}
	return nil
}
