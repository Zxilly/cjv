package settings

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"

	"github.com/Zxilly/cjv/internal/config"
	"github.com/Zxilly/cjv/internal/i18n"
	"github.com/Zxilly/cjv/internal/toolchain"
	"github.com/spf13/cobra"
)

var overrideSetPath string

var overrideCmd = &cobra.Command{
	Use:   "override",
	Short: "Manage directory toolchain overrides",
}

var overrideSetCmd = &cobra.Command{
	Use:   "set <toolchain>",
	Short: "Set a toolchain override for the current or specified directory",
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

		dir, err := resolveOverrideDir(overrideSetPath)
		if err != nil {
			return err
		}

		sf, settings, err := LoadSettings()
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
		if err := sf.Save(settings); err != nil {
			return err
		}

		fmt.Println(i18n.T("OverrideSet", i18n.MsgData{
			"Dir":       dir,
			"Toolchain": tc,
		}))
		return nil
	},
}

var overrideUnsetPath string
var overrideUnsetNonexistent bool

var overrideUnsetCmd = &cobra.Command{
	Use:   "unset",
	Short: "Remove the toolchain override for the current or specified directory",
	RunE: func(cmd *cobra.Command, args []string) error {
		sf, settings, err := LoadSettings()
		if err != nil {
			return err
		}

		if overrideUnsetNonexistent {
			return unsetNonexistentOverrides(settings, sf)
		}

		dir, err := resolveOverrideDir(overrideUnsetPath)
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
			return fmt.Errorf("no override set for %s", dir)
		}

		if err := sf.Save(settings); err != nil {
			return err
		}

		fmt.Println(i18n.T("OverrideUnset", i18n.MsgData{"Dir": dir}))
		return nil
	},
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

func unsetNonexistentOverrides(settings *config.Settings, sf *config.SettingsFile) error {
	removed := 0
	for dir := range settings.Overrides {
		if _, err := os.Stat(dir); errors.Is(err, os.ErrNotExist) {
			delete(settings.Overrides, dir)
			removed++
		}
	}
	if removed == 0 {
		fmt.Println(i18n.T("NoNonexistentOverrides", nil))
		return nil
	}
	if err := sf.Save(settings); err != nil {
		return err
	}
	fmt.Println(i18n.TP("RemovedNonexistentOverrides", i18n.MsgData{"Count": strconv.Itoa(removed)}, removed))
	return nil
}

var overrideListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all directory overrides",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, settings, err := LoadSettings()
		if err != nil {
			return err
		}

		if len(settings.Overrides) == 0 {
			fmt.Println(i18n.T("NoOverrides", nil))
			return nil
		}

		dirs := make([]string, 0, len(settings.Overrides))
		for dir := range settings.Overrides {
			dirs = append(dirs, dir)
		}
		slices.Sort(dirs)
		for _, dir := range dirs {
			fmt.Printf("%-50s → %s\n", dir, settings.Overrides[dir])
		}
		return nil
	},
}
