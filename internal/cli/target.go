package cli

import (
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

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
			label = " (installed)"
		}
		fmt.Fprintf(&b, "%s%s\n", target.Name, label)
	}
	return b.String()
}

func (app *application) initTargetCommands() {
	var selected string
	var installedOnly bool
	command := &cobra.Command{Use: "target", Short: "Manage cross SDKs for a host toolchain", Args: cobra.NoArgs}
	command.PersistentFlags().StringVar(&selected, "toolchain", "", "Host toolchain (defaults to the active toolchain)")
	resolveHost := func() (string, toolchain.ToolchainName, toolchain.Installation, error) {
		input, err := app.selectedToolchain(selected)
		if err != nil {
			return "", toolchain.ToolchainName{}, toolchain.Installation{}, err
		}
		dir, _, identity, err := toolchain.FindActiveDir(input)
		if err != nil {
			return "", identity, toolchain.Installation{}, err
		}
		if identity.IsCustom() {
			return "", identity, toolchain.Installation{}, fmt.Errorf("custom toolchain %s has no published targets", input)
		}
		identity, err = toolchain.ParseToolchainName(filepath.Base(dir))
		if err != nil {
			return "", identity, toolchain.Installation{}, err
		}
		if identity.IsCustom() {
			return "", identity, toolchain.Installation{}, fmt.Errorf("custom toolchain %s has no published targets", input)
		}
		record, err := toolchain.ReadInstallation(dir)
		if err != nil {
			return "", identity, record, err
		}
		if record.Tuple == "" {
			d, err := lifecycle.OpenDistribution(app.lifecycleOptions())
			if err != nil {
				return "", identity, record, err
			}
			record.Tuple = d.HostTuple
		}
		return dir, identity, record, nil
	}
	list := &cobra.Command{Use: "list", Short: "List published or installed targets", Args: cobra.NoArgs, RunE: func(cmd *cobra.Command, args []string) error {
		dir, identity, record, err := resolveHost()
		if err != nil {
			return err
		}
		release, err := toolchain.ParseToolchainName(record.Release)
		if err != nil {
			return err
		}
		installed, err := toolchain.ListInstalled()
		if err != nil {
			return err
		}
		found := map[string]bool{}
		for _, name := range installed {
			p, err := toolchain.ParseToolchainName(name)
			if err != nil || p.Channel != identity.Channel || p.Version != identity.Version || !strings.HasPrefix(p.Target, record.Tuple+"-") {
				continue
			}
			root, err := toolchain.FindInstalled(p)
			if err != nil {
				return err
			}
			current, err := toolchain.InstalledRelease(root)
			if err != nil {
				return err
			}
			if current.Version == release.Version {
				found[strings.TrimPrefix(p.Target, record.Tuple+"-")] = true
			}
		}
		if !installedOnly {
			d, err := lifecycle.OpenDistribution(app.lifecycleOptions())
			if err != nil {
				return err
			}
			names, err := d.Source.AvailableTargets(cmd.Context(), release.Channel, release.Version, record.Tuple)
			if err != nil {
				return err
			}
			for _, name := range names {
				if _, ok := found[name]; !ok {
					found[name] = false
				}
			}
		}
		result := targetListResult{Toolchain: filepath.Base(dir), Targets: []targetEntry{}}
		var names []string
		for name := range found {
			names = append(names, name)
		}
		slices.Sort(names)
		for _, name := range names {
			result.Targets = append(result.Targets, targetEntry{Name: name, Installed: found[name]})
		}
		return app.output.RenderTo(cmdOutput(cmd), result)
	}}
	list.Flags().BoolVar(&installedOnly, "installed", false, "List only installed targets without accessing the distribution")
	add := &cobra.Command{Use: "add <target>...", Short: "Install targets for the selected host release", Args: cobra.MinimumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		dir, _, _, err := resolveHost()
		if err != nil {
			return err
		}
		if err := lifecycle.Install(cmd.Context(), lifecycle.InstallRequest{Toolchain: filepath.Base(dir), Targets: args, NoUpdate: true}, app.lifecycleOptions()); err != nil {
			return err
		}
		return app.output.RenderTo(cmdOutput(cmd), installResult{Toolchain: filepath.Base(dir), Targets: args})
	}}
	remove := &cobra.Command{Use: "remove <target>...", Aliases: []string{"uninstall"}, Short: "Remove targets without removing their host", Args: cobra.MinimumNArgs(1), RunE: func(cmd *cobra.Command, args []string) error {
		_, identity, record, err := resolveHost()
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
			name := identity
			name.Host = ""
			name.Target = record.Tuple + "-" + target
			if err := lifecycle.RemoveToolchain(name.String()); err != nil {
				errs = append(errs, err)
				continue
			}
			removed = append(removed, uninstallResult{Name: name.String()})
		}
		return app.output.RenderOutcome(cmdOutput(cmd), uninstallBatchResult{Removed: removed}, errors.Join(errs...))
	}}
	command.AddCommand(list, add, remove)
	app.rootCmd.AddCommand(command)
}
