// Package projectcmd implements project pinning and read-only diagnostics.
package projectcmd

import (
	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/spf13/cobra"
)

func Register(root *cobra.Command, shared *clictx.Context) {
	root.AddCommand(newPinCommand(shared), newDoctorCommand(shared))
}
