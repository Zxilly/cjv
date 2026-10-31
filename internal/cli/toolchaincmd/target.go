package toolchaincmd

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/lifecycle"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
)

type targetEntry struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
}
type targetListResult struct {
	Toolchain string        `json:"toolchain"`
	Targets   []targetEntry `json:"targets"`
}

func (r targetListResult) Text() string {
	var b strings.Builder
	for _, target := range r.Targets {
		label := ""
		if target.Installed {
			label = i18n.T("TargetInstalledLabel", nil)
		}
		fmt.Fprintf(&b, "%s%s\n", target.Name, label)
	}
	return b.String()
}

func targetCommand(ctx *clictx.Context) *cobra.Command {
	var selected string
	var installedOnly bool
	command := &cobra.Command{Use: "target", Short: i18n.T("TargetCmdShort", nil), Args: cobra.NoArgs}
	command.PersistentFlags().StringVar(&selected, "toolchain", "", i18n.T("TargetFlagToolchain", nil))
	resolveHost := func() (toolchain.HostTargets, error) {
		input, err := ctx.SelectedToolchain(selected)
		if err != nil {
			return toolchain.HostTargets{}, err
		}
		dir, _, identity, err := toolchain.FindActiveDir(input)
		if err != nil {
			return toolchain.HostTargets{}, err
		}
		if identity.IsCustom() {
			return toolchain.HostTargets{}, fmt.Errorf(i18n.T("TargetCustomUnavailable", nil), input)
		}
		return toolchain.ReadHostTargets(dir)
	}
	list := &cobra.Command{Use: "list", Short: i18n.T("TargetListShort", nil), Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		host, err := resolveHost()
		if err != nil {
			return err
		}
		installed, err := host.Installed()
		if err != nil {
			return err
		}
		found := map[string]bool{}
		for _, name := range installed {
			found[name] = true
		}
		if !installedOnly {
			d, err := lifecycle.OpenDistribution(ctx.LifecycleOptions())
			if err != nil {
				return err
			}
			names, err := d.Source.AvailableTargets(cmd.Context(), host.Release.Channel, host.Release.Version, host.Tuple)
			if err != nil {
				return err
			}
			for _, name := range names {
				if _, ok := found[name]; !ok {
					found[name] = false
				}
			}
		}
		result := targetListResult{Toolchain: filepath.Base(host.Dir), Targets: []targetEntry{}}
		var names []string
		for name := range found {
			names = append(names, name)
		}
		slices.Sort(names)
		for _, name := range names {
			result.Targets = append(result.Targets, targetEntry{Name: name, Installed: found[name]})
		}
		return ctx.Output.RenderTo(clictx.Out(cmd), result)
	}}
	list.Flags().BoolVar(&installedOnly, "installed", false, i18n.T("TargetFlagInstalled", nil))
	add := &cobra.Command{Use: "add <target>...", Short: i18n.T("TargetAddShort", nil), Args: cobra.MinimumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		host, err := resolveHost()
		if err != nil {
			return err
		}
		if err := lifecycle.Install(cmd.Context(), lifecycle.InstallRequest{Toolchain: filepath.Base(host.Dir), Targets: args, NoUpdate: true}, ctx.LifecycleOptions()); err != nil {
			return err
		}
		return ctx.Output.RenderTo(clictx.Out(cmd), installResult{Toolchain: filepath.Base(host.Dir), Targets: args})
	}}
	remove := &cobra.Command{Use: "remove <target>...", Aliases: []string{"uninstall"}, Short: i18n.T("TargetRemoveShort", nil), Args: cobra.MinimumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		host, err := resolveHost()
		if err != nil {
			return err
		}
		targets, err := sdktarget.NormalizeList(args)
		if err != nil {
			return err
		}
		var errs []error
		var removed []uninstallResult
		for _, target := range targets {
			name, err := host.TargetName(target)
			if err != nil {
				errs = append(errs, err)
				continue
			}
			if err := lifecycle.RemoveToolchain(name.String()); err != nil {
				errs = append(errs, err)
				continue
			}
			removed = append(removed, uninstallResult{Name: name.String()})
		}
		return ctx.Output.RenderOutcome(clictx.Out(cmd), uninstallBatchResult{Removed: removed}, errors.Join(errs...))
	}}
	command.AddCommand(list, add, remove)
	return command
}
