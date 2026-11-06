package settingscmd

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
)

func newOverrideCommand(ctx *clictx.Context) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "override",
		Short: i18n.T("OverrideCmdShort", nil),
	}
	cmd.AddCommand(newOverrideSetCommand(ctx), newOverrideUnsetCommand(ctx), newOverrideListCommand(ctx))
	return cmd
}

type overrideEntry struct {
	Directory string `json:"directory"`
	Toolchain string `json:"toolchain"`
}

func (r overrideEntry) Text() string {
	return i18n.T("OverrideSet", i18n.MsgData{"Dir": r.Directory, "Toolchain": r.Toolchain})
}

type overrideListResult struct {
	Overrides []overrideEntry `json:"overrides"`
}

func (r overrideListResult) Text() string {
	if len(r.Overrides) == 0 {
		return i18n.T("NoOverrides", nil)
	}
	var b strings.Builder
	for _, entry := range r.Overrides {
		fmt.Fprintf(&b, "%-50s → %s\n", entry.Directory, entry.Toolchain)
	}
	return b.String()
}

type overrideRemoveResult struct {
	Removed     []string `json:"removed"`
	nonexistent bool
}

func (r overrideRemoveResult) Text() string {
	if r.nonexistent {
		if len(r.Removed) == 0 {
			return i18n.T("NoNonexistentOverrides", nil)
		}
		return i18n.TP("RemovedNonexistentOverrides", i18n.MsgData{"Count": strconv.Itoa(len(r.Removed))}, len(r.Removed))
	}
	var b strings.Builder
	for _, dir := range r.Removed {
		b.WriteString(i18n.T("OverrideUnset", i18n.MsgData{"Dir": dir}))
		b.WriteByte('\n')
	}
	return b.String()
}

func newOverrideSetCommand(ctx *clictx.Context) *cobra.Command {
	var path string
	cmd := &cobra.Command{
		Use:   "set <toolchain>",
		Short: i18n.T("OverrideSetShort", nil),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tc := args[0]

			// Validate and normalize toolchain name
			parsed, err := toolchain.ParseActiveName(tc)
			if err != nil {
				return err
			}
			// Normalize standard names; accept custom names as-is
			if parsed.Channel != toolchain.UnknownChannel {
				tc = parsed.String()
			}

			dir, err := resolveOverrideDir(path)
			if err != nil {
				return err
			}

			sf, err := config.DefaultSettingsFile()
			if err != nil {
				return err
			}

			if _, err := sf.Mutate(func(settings *config.Settings) (config.SettingsUpdate, error) {
				changes := map[string]*string{dir: &tc}
				// Normalize aliases while holding the settings file lock.
				for key := range settings.Overrides {
					if config.NormalizePath(key) == dir && key != dir {
						changes[key] = nil
					}
				}
				return config.SettingsUpdate{OverrideChanges: changes}, nil
			}); err != nil {
				return err
			}

			return ctx.Output.RenderTo(cmd.OutOrStdout(), overrideEntry{Directory: dir, Toolchain: tc})
		},
	}
	cmd.Flags().StringVar(&path, "path", "", i18n.T("OverrideFlagPath", nil))
	return cmd
}

func newOverrideUnsetCommand(ctx *clictx.Context) *cobra.Command {
	var path string
	var nonexistent bool
	cmd := &cobra.Command{
		Use:   "unset",
		Short: i18n.T("OverrideUnsetShort", nil),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, err := config.DefaultSettingsFile()
			if err != nil {
				return err
			}

			if nonexistent {
				removed, err := removeNonexistentOverrides(sf)
				if err != nil {
					return err
				}
				return ctx.Output.RenderTo(cmd.OutOrStdout(), overrideRemoveResult{Removed: removed, nonexistent: true})
			}

			dir, err := resolveOverrideDir(path)
			if err != nil {
				return err
			}

			if _, err := sf.Mutate(func(settings *config.Settings) (config.SettingsUpdate, error) {
				changes := make(map[string]*string)
				for key := range settings.Overrides {
					if config.NormalizePath(key) == dir {
						changes[key] = nil
					}
				}
				if len(changes) == 0 {
					return config.SettingsUpdate{}, errors.New(i18n.T("NoOverrideSet", i18n.MsgData{"Dir": dir}))
				}
				return config.SettingsUpdate{OverrideChanges: changes}, nil
			}); err != nil {
				return err
			}

			return ctx.Output.RenderTo(cmd.OutOrStdout(), overrideRemoveResult{Removed: []string{dir}})
		},
	}
	cmd.Flags().StringVar(&path, "path", "", i18n.T("OverrideUnsetFlagPath", nil))
	cmd.Flags().BoolVar(&nonexistent, "nonexistent", false, i18n.T("OverrideFlagNonexistent", nil))
	cmd.MarkFlagsMutuallyExclusive("path", "nonexistent")
	return cmd
}

// resolveOverrideDir returns the normalized directory for override operations.
// Uses flagPath if provided, otherwise the current working directory.
func resolveOverrideDir(flagPath string) (string, error) {
	dir := flagPath
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	return config.NormalizePath(dir), nil
}

func removeNonexistentOverrides(sf *config.SettingsFile) ([]string, error) {
	removed := make([]string, 0)
	_, err := sf.Mutate(func(settings *config.Settings) (config.SettingsUpdate, error) {
		changes := make(map[string]*string)
		for dir := range settings.Overrides {
			if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
				changes[dir] = nil
				removed = append(removed, dir)
			}
		}
		return config.SettingsUpdate{OverrideChanges: changes}, nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(removed)
	return removed, nil
}

func newOverrideListCommand(ctx *clictx.Context) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: i18n.T("OverrideListShort", nil),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, settings, err := config.LoadDefaultSettings()
			if err != nil {
				return err
			}

			dirs := make([]string, 0, len(settings.Overrides))
			for dir := range settings.Overrides {
				dirs = append(dirs, dir)
			}
			slices.Sort(dirs)
			entries := make([]overrideEntry, 0, len(dirs))
			for _, dir := range dirs {
				entries = append(entries, overrideEntry{Directory: dir, Toolchain: settings.Overrides[dir]})
			}
			return ctx.Output.RenderTo(cmd.OutOrStdout(), overrideListResult{Overrides: entries})
		},
	}
}
