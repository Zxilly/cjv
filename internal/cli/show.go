package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Zxilly/cjv/internal/cjverr"
	"github.com/Zxilly/cjv/internal/cli/output"
	componentlib "github.com/Zxilly/cjv/internal/component"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
)

var showCmd = &cobra.Command{
	Use:   "show",
	Short: "Show active and installed toolchains",
	RunE:  runShowDefault,
}

var showActiveCmd = &cobra.Command{
	Use:   "active",
	Short: "Show the active toolchain",
	RunE:  runShowActive,
}

var showInstalledCmd = &cobra.Command{
	Use:   "installed",
	Short: "List installed toolchains",
	RunE:  runShowInstalled,
}

var showHomeCmd = &cobra.Command{
	Use:   "home",
	Short: "Show CJV_HOME path",
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := config.Home()
		if err != nil {
			return err
		}
		return output.RenderTo(cmdOutput(cmd), showHomeResult{Home: home})
	},
}

type showHomeResult struct {
	Home string `json:"home"`
}

func (r showHomeResult) Text() string { return r.Home }

type showActiveResult struct {
	Name      string `json:"name"`
	Source    string `json:"source"`
	Installed bool   `json:"installed"`
}

func (r showActiveResult) Text() string {
	name := r.Name
	if !r.Installed {
		name = name + " (not installed)"
	}
	return i18n.T("ActiveToolchain", i18n.MsgData{
		"Name":   name,
		"Source": r.Source,
	})
}

type installedEntry struct {
	Name       string   `json:"name"`
	Components []string `json:"components"`
}

type showInstalledResult struct {
	Toolchains []installedEntry `json:"toolchains"`
}

func (r showInstalledResult) Text() string {
	if len(r.Toolchains) == 0 {
		return i18n.T("NoToolchainsInstalled", nil)
	}
	var b strings.Builder
	b.WriteString(i18n.TP("InstalledToolchains", i18n.MsgData{
		"Count": strconv.Itoa(len(r.Toolchains)),
	}, len(r.Toolchains)))
	b.WriteByte('\n')
	for _, e := range r.Toolchains {
		fmt.Fprintf(&b, "  %s\n", e.Name)
		if len(e.Components) > 0 {
			fmt.Fprintf(&b, "    %s %s\n",
				i18n.TP("InstalledComponents",
					i18n.MsgData{"Count": strconv.Itoa(len(e.Components))},
					len(e.Components)),
				strings.Join(e.Components, ", "))
		}
	}
	return b.String()
}

type showDefaultResult struct {
	Active      *showActiveResult   `json:"active,omitempty"`
	DefaultHost *string             `json:"default_host,omitempty"`
	Installed   showInstalledResult `json:"installed"`
}

func (r showDefaultResult) Text() string {
	var b strings.Builder
	if r.Active != nil {
		b.WriteString(r.Active.Text())
		b.WriteByte('\n')
	}
	if r.DefaultHost != nil {
		host := *r.DefaultHost
		if host == "" {
			host = i18n.T("ShowDefaultHostAuto", nil)
		}
		b.WriteString(i18n.T("ShowDefaultHost", i18n.MsgData{"Host": host}))
		b.WriteByte('\n')
	}
	b.WriteByte('\n')
	b.WriteString(r.Installed.Text())
	return b.String()
}

func gatherActive() (*showActiveResult, error) {
	_, name, source, err := toolchain.ResolveActiveToolchain()
	if err != nil {
		if notInstalled, ok := errors.AsType[*cjverr.ToolchainNotInstalledError](err); ok {
			return &showActiveResult{
				Name:   notInstalled.Name,
				Source: source.String(),
			}, nil
		}
		return nil, err
	}
	return &showActiveResult{Name: name, Source: source.String(), Installed: true}, nil
}

func gatherInstalled() (showInstalledResult, error) {
	installed, err := toolchain.ListInstalled()
	if err != nil {
		return showInstalledResult{}, err
	}
	if len(installed) == 0 {
		return showInstalledResult{}, nil
	}
	tcRoot, err := config.ToolchainsDir()
	if err != nil {
		return showInstalledResult{}, err
	}
	entries := make([]installedEntry, 0, len(installed))
	for _, name := range installed {
		entry := installedEntry{Name: name, Components: []string{}}
		if comps, err := componentlib.ListInstalled(filepath.Join(tcRoot, name)); err == nil && len(comps) > 0 {
			parts := make([]string, len(comps))
			for i, c := range comps {
				parts[i] = string(c)
			}
			entry.Components = parts
		}
		entries = append(entries, entry)
	}
	return showInstalledResult{Toolchains: entries}, nil
}

func runShowDefault(cmd *cobra.Command, args []string) error {
	active, activeErr := gatherActive()
	if activeErr != nil {
		if _, ok := errors.AsType[*cjverr.NoToolchainConfiguredError](activeErr); ok {
			// Surface the informative message on stderr for humans, but do
			// not error out — the rest of the report is still useful.
			if !output.IsJSON() {
				fmt.Fprintln(os.Stderr, activeErr)
			}
			active = nil
		} else {
			return activeErr
		}
	}

	var defaultHost *string
	if sf, err := config.DefaultSettingsFile(); err == nil {
		if settings, err := sf.Load(); err == nil && settings != nil {
			host := settings.DefaultHost
			defaultHost = &host
		}
	}

	installed, err := gatherInstalled()
	if err != nil {
		return err
	}

	return output.RenderTo(cmdOutput(cmd), showDefaultResult{
		Active:      active,
		DefaultHost: defaultHost,
		Installed:   installed,
	})
}

func runShowActive(cmd *cobra.Command, args []string) error {
	active, err := gatherActive()
	if err != nil {
		return err
	}
	return output.RenderTo(cmdOutput(cmd), *active)
}

func runShowInstalled(cmd *cobra.Command, args []string) error {
	r, err := gatherInstalled()
	if err != nil {
		return err
	}
	return output.RenderTo(cmdOutput(cmd), r)
}

func init() {
	showCmd.AddCommand(showActiveCmd)
	showCmd.AddCommand(showInstalledCmd)
	showCmd.AddCommand(showHomeCmd)
	rootCmd.AddCommand(showCmd)
}
