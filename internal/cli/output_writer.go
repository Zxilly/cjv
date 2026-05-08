package cli

import (
	"io"
	"os"

	"github.com/spf13/cobra"
)

func cmdOutput(cmd *cobra.Command) io.Writer {
	if cmd == nil {
		return os.Stdout
	}
	return cmd.OutOrStdout()
}
