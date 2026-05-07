package settings

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestRegisterCommandsAddsSettingsCommands(t *testing.T) {
	root := &cobra.Command{Use: "cjv"}

	RegisterCommands(root)

	var names []string
	for _, cmd := range root.Commands() {
		names = append(names, cmd.Name())
	}
	assert.Contains(t, names, "default")
	assert.Contains(t, names, "override")
	assert.Contains(t, names, "set")
}
