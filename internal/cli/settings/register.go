package settings

import "github.com/spf13/cobra"

// RegisterCommands adds all settings-related subcommands to the given root command.
func RegisterCommands(root *cobra.Command) {
	root.AddCommand(newSetCommand(), newDefaultCommand(), newOverrideCommand())
}
