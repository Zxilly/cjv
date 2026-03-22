package settings

import "github.com/spf13/cobra"

// RegisterCommands adds all settings-related subcommands to the given root command.
func RegisterCommands(root *cobra.Command) {
	// set
	setCmd.AddCommand(setAutoSelfUpdateCmd)
	setCmd.AddCommand(setAutoInstallCmd)
	setCmd.AddCommand(setDefaultHostCmd)
	setCmd.AddCommand(setProfileCmd)
	root.AddCommand(setCmd)

	// default
	root.AddCommand(defaultCmd)

	// override
	overrideSetCmd.Flags().StringVar(&overrideSetPath, "path", "", "Set override for specified directory instead of current directory")
	overrideUnsetCmd.Flags().StringVar(&overrideUnsetPath, "path", "", "Unset override for specified directory instead of current directory")
	overrideUnsetCmd.Flags().BoolVar(&overrideUnsetNonexistent, "nonexistent", false, "Remove all overrides for directories that no longer exist")
	overrideCmd.AddCommand(overrideSetCmd)
	overrideCmd.AddCommand(overrideUnsetCmd)
	overrideCmd.AddCommand(overrideListCmd)
	root.AddCommand(overrideCmd)
}
