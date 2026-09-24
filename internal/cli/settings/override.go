package settings

import (
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strconv"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
)

func newOverrideCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "override",
		Short: i18n.T("OverrideCmdShort", nil),
	}
	cmd.AddCommand(newOverrideSetCommand(), newOverrideUnsetCommand(), newOverrideListCommand())
	return cmd
}

func newOverrideSetCommand() *cobra.Command {
	var path string
	cmd := &cobra.Command{
		Use:   "set <toolchain>",
		Short: i18n.T("OverrideSetShort", nil),
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			tc := args[0]

			// Validate and normalize toolchain name
			parsed, err := toolchain.ParseToolchainName(tc)
			if err != nil {
				return err
			}
			if err := ensureActiveToolchainName(tc, parsed); err != nil {
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

			sf, settings, err := config.LoadDefaultSettings()
			if err != nil {
				return err
			}

			// Remove any existing entry whose normalized path matches dir
			// to prevent duplicate entries from different string representations.
			for key := range settings.Overrides {
				if config.NormalizePath(key) == dir && key != dir {
					delete(settings.Overrides, key)
				}
			}
			settings.Overrides[dir] = tc
			if _, err := sf.Update(config.SettingsUpdate{Overrides: settings.Overrides}); err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), i18n.T("OverrideSet", i18n.MsgData{
				"Dir":       dir,
				"Toolchain": tc,
			}))
			return err
		},
	}
	cmd.Flags().StringVar(&path, "path", "", "Set override for specified directory instead of current directory")
	return cmd
}

func newOverrideUnsetCommand() *cobra.Command {
	var path string
	var nonexistent bool
	cmd := &cobra.Command{
		Use:   "unset",
		Short: i18n.T("OverrideUnsetShort", nil),
		RunE: func(cmd *cobra.Command, args []string) error {
			sf, settings, err := config.LoadDefaultSettings()
			if err != nil {
				return err
			}

			if nonexistent {
				return unsetNonexistentOverrides(cmd.OutOrStdout(), settings, sf)
			}

			dir, err := resolveOverrideDir(path)
			if err != nil {
				return err
			}

			found := false
			for key := range settings.Overrides {
				if config.NormalizePath(key) == dir {
					delete(settings.Overrides, key)
					found = true
					break
				}
			}
			if !found {
				return errors.New(i18n.T("NoOverrideSet", i18n.MsgData{"Dir": dir}))
			}

			if _, err := sf.Update(config.SettingsUpdate{Overrides: settings.Overrides}); err != nil {
				return err
			}

			_, err = fmt.Fprintln(cmd.OutOrStdout(), i18n.T("OverrideUnset", i18n.MsgData{"Dir": dir}))
			return err
		},
	}
	cmd.Flags().StringVar(&path, "path", "", "Unset override for specified directory instead of current directory")
	cmd.Flags().BoolVar(&nonexistent, "nonexistent", false, "Remove all overrides for directories that no longer exist")
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

func unsetNonexistentOverrides(w io.Writer, settings *config.Settings, sf *config.SettingsFile) error {
	removed := 0
	for dir := range settings.Overrides {
		if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
			delete(settings.Overrides, dir)
			removed++
		}
	}
	if removed == 0 {
		_, err := fmt.Fprintln(w, i18n.T("NoNonexistentOverrides", nil))
		return err
	}
	if _, err := sf.Update(config.SettingsUpdate{Overrides: settings.Overrides}); err != nil {
		return err
	}
	_, err := fmt.Fprintln(w, i18n.TP("RemovedNonexistentOverrides", i18n.MsgData{"Count": strconv.Itoa(removed)}, removed))
	return err
}

func newOverrideListCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: i18n.T("OverrideListShort", nil),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, settings, err := config.LoadDefaultSettings()
			if err != nil {
				return err
			}

			if len(settings.Overrides) == 0 {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), i18n.T("NoOverrides", nil))
				return err
			}

			dirs := make([]string, 0, len(settings.Overrides))
			for dir := range settings.Overrides {
				dirs = append(dirs, dir)
			}
			slices.Sort(dirs)
			for _, dir := range dirs {
				if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%-50s → %s\n", dir, settings.Overrides[dir]); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
