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

type componentEntry struct {
	Name      string `json:"name"`
	Installed bool   `json:"installed"`
}

type componentListResult struct {
	Toolchain  string           `json:"toolchain"`
	Components []componentEntry `json:"components"`
}

// componentListView attaches presentation flags to a componentListResult so
// they can affect Text() rendering without polluting the JSON data shape.
type componentListView struct {
	componentListResult
	quiet         bool
	installedOnly bool
}

func (v componentListView) Text() string {
	if v.quiet {
		var b strings.Builder
		for _, c := range v.Components {
			b.WriteString(c.Name)
			b.WriteByte('\n')
		}
		return b.String()
	}
	if len(v.Components) == 0 && v.installedOnly {
		return i18n.T("NoComponentsInstalled", nil)
	}
	var b strings.Builder
	for _, c := range v.Components {
		label := i18n.T("AvailableComponents", nil)
		if c.Installed {
			label = i18n.T("StatusInstalled", nil)
		}
		fmt.Fprintf(&b, "%s (%s)\n", c.Name, label)
	}
	return b.String()
}

func (v componentListView) JSONValue() any {
	return v.componentListResult
}

type componentRemovedEntry struct {
	Toolchain string `json:"toolchain"`
	Component string `json:"component"`
}

type componentRemoveResult struct {
	Removed []componentRemovedEntry `json:"removed"`
}

type componentAddResult struct {
	Toolchain  string              `json:"toolchain"`
	Components []componentlib.Name `json:"components"`
	Forced     bool                `json:"forced"`
}

func (r componentAddResult) Text() string { return "" }

func (r componentRemoveResult) Text() string {
	var b strings.Builder
	for _, e := range r.Removed {
		b.WriteString(color.GreenString(i18n.T("ComponentRemoved", i18n.MsgData{
			"Toolchain": e.Toolchain,
			"Component": e.Component,
		})))
		b.WriteByte('\n')
	}
	return b.String()
}

type componentLinkResult struct {
	Toolchain string `json:"toolchain"`
	Component string `json:"component"`
	Path      string `json:"path"`
}

func (r componentLinkResult) Text() string {
	return color.GreenString(i18n.T("ComponentLinked", i18n.MsgData{
		"Component": r.Component,
		"Path":      r.Path,
		"Toolchain": r.Toolchain,
	}))
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

func (app *application) runComponentAdd(cmd *cobra.Command, args []string) error {
	tcDir, tcName, err := resolveToolchainArg(app.componentToolchain)
	if err != nil {
		return err
	}
	if tcName.IsCustom() {
		return &cjverr.ComponentRequiresHostError{Component: args[0]}
	}
	if err := app.installComponentsList(cmd.Context(), filepath.Base(tcDir), args, app.componentAddForce, false); err != nil {
		return err
	}
	components, err := componentlib.NormalizeList(args)
	if err != nil {
		return err
	}
	return app.output.RenderTo(cmdOutput(cmd), componentAddResult{
		Toolchain:  filepath.Base(tcDir),
		Components: components,
		Forced:     app.componentAddForce,
	})
}

func (app *application) runComponentRemove(cmd *cobra.Command, args []string) error {
	parsed, parseErrs := parseComponentRemoveArgs(args)
	if len(parsed) == 0 {
		return errors.Join(parseErrs...)
	}

	tcDir, tcName, err := resolveToolchainArg(app.componentToolchain)
	if err != nil {
		return err
	}
	toolchainName := tcName.String()

	roots, err := componentlib.RootsFor(toolchainName)
	if err != nil {
		return err
	}

	var removeErrs []error
	removeErrs = append(removeErrs, parseErrs...)
	result := componentRemoveResult{}
	for _, c := range parsed {
		if !componentlib.IsInstalled(tcDir, c) {
			removeErrs = append(removeErrs, &cjverr.ComponentNotInstalledError{
				Toolchain: toolchainName,
				Component: string(c),
			})
			continue
		}
		if !app.output.IsJSON() {
			fmt.Println(i18n.T("RemovingComponent", i18n.MsgData{"Component": string(c)}))
		}
		if err := componentlib.Remove(roots, c); err != nil {
			removeErrs = append(removeErrs, err)
			continue
		}
		result.Removed = append(result.Removed, componentRemovedEntry{
			Toolchain: toolchainName,
			Component: string(c),
		})
	}
	joinErr := errors.Join(removeErrs...)
	// Emit what was actually removed before surfacing any error so a partial
	// removal is visible to the caller. In JSON mode the root handler writes an
	// error envelope to stdout on failure, so only render the result there when
	// it succeeded to avoid emitting two JSON documents.
	if joinErr == nil || !app.output.IsJSON() {
		if renderErr := app.output.RenderTo(cmdOutput(cmd), result); renderErr != nil {
			return errors.Join(joinErr, renderErr)
		}
	}
	return joinErr
}

func (app *application) runComponentLink(cmd *cobra.Command, args []string) error {
	name, err := componentlib.ParseName(args[0])
	if err != nil {
		return err
	}

	tcDir, _, err := resolveToolchainArg(app.componentToolchain)
	if err != nil {
		return err
	}
	toolchainName := filepath.Base(tcDir)

	roots, err := componentlib.RootsFor(toolchainName)
	if err != nil {
		return err
	}

	if !app.output.IsJSON() {
		fmt.Println(i18n.T("InstallingComponent", i18n.MsgData{"Component": string(name)}))
	}

	absPath, err := app.componentLinkFunc(roots, name, args[1], app.componentLinkForce)
	if err != nil {
		return err
	}

	return app.output.RenderTo(cmdOutput(cmd), componentLinkResult{
		Toolchain: toolchainName,
		Component: string(name),
		Path:      absPath,
	})
}

func (app *application) runComponentList(cmd *cobra.Command, args []string) error {
	tcDir, tcName, err := resolveToolchainArg(app.componentToolchain)
	if err != nil {
		return err
	}

	installed, err := componentlib.ListInstalled(tcDir)
	if err != nil {
		return err
	}

	var available []componentlib.Name
	if !tcName.IsCustom() {
		tuple := tcName.Target
		if tuple == "" {
			_, settings, err := clisettings.LoadSettings()
			if err != nil {
				return err
			}
			tuple, err = dist.CurrentHostTuple(settings.DefaultHost)
			if err != nil {
				return err
			}
		}
		available = componentlib.AvailableComponents(tcName, tuple)
	}

	// Initialize Components to a non-nil empty slice so `--json` emits [] rather
	// than null for the empty state, keeping the array field's shape stable.
	result := componentListResult{Toolchain: tcName.String(), Components: []componentEntry{}}
	for _, n := range installed {
		result.Components = append(result.Components, componentEntry{Name: string(n), Installed: true})
	}
	if !app.componentListInstalledOnly {
		for _, n := range available {
			if slices.Contains(installed, n) {
				continue
			}
			result.Components = append(result.Components, componentEntry{Name: string(n), Installed: false})
		}
	}
	return app.output.RenderTo(cmdOutput(cmd), componentListView{
		componentListResult: result,
		quiet:               app.componentListQuiet,
		installedOnly:       app.componentListInstalledOnly,
	})
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

func (app *application) initComponentCommands() {
	app.componentLinkFunc = componentlib.Link
	app.componentCmd = &cobra.Command{
		Use:   "component",
		Short: i18n.T("ComponentSubcmdShort", nil),
		Args:  cobra.NoArgs,
	}
	app.componentAddCmd = &cobra.Command{
		Use:   "add <name>...",
		Short: i18n.T("ComponentAddShort", nil),
		Args:  cobra.MinimumNArgs(1),
		RunE:  app.runComponentAdd,
	}
	app.componentRemoveCmd = &cobra.Command{
		Use:     "remove <name>...",
		Aliases: []string{"uninstall", "rm", "delete", "del"},
		Short:   i18n.T("ComponentRemoveShort", nil),
		Args:    cobra.MinimumNArgs(1),
		RunE:    app.runComponentRemove,
	}
	app.componentListCmd = &cobra.Command{
		Use:   "list",
		Short: i18n.T("ComponentListShort", nil),
		Args:  cobra.NoArgs,
		RunE:  app.runComponentList,
	}
	app.componentLinkCmd = &cobra.Command{
		Use:   "link <name> <path>",
		Short: i18n.T("ComponentLinkShort", nil),
		Args:  cobra.ExactArgs(2),
		RunE:  app.runComponentLink,
	}

	app.componentCmd.PersistentFlags().StringVar(&app.componentToolchain, "toolchain", "", i18n.T("ComponentFlagToolchain", nil))
	app.componentAddCmd.Flags().BoolVar(&app.componentAddForce, "force", false, i18n.T("InstallFlagForce", nil))
	app.componentListCmd.Flags().BoolVar(&app.componentListInstalledOnly, "installed", false, i18n.T("ComponentFlagInstalled", nil))
	app.componentListCmd.Flags().BoolVarP(&app.componentListQuiet, "quiet", "q", false, i18n.T("ComponentFlagQuiet", nil))
	app.componentLinkCmd.Flags().BoolVar(&app.componentLinkForce, "force", false, i18n.T("ComponentLinkFlagForce", nil))

	app.componentCmd.AddCommand(app.componentAddCmd)
	app.componentCmd.AddCommand(app.componentRemoveCmd)
	app.componentCmd.AddCommand(app.componentListCmd)
	app.componentCmd.AddCommand(app.componentLinkCmd)
	app.rootCmd.AddCommand(app.componentCmd)

}
