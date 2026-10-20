package cli

import (
	"bytes"
	"testing"

	"github.com/Zxilly/cjv/internal/cli/clitest"
)

func TestMain(m *testing.M) { clitest.Main(m) }

// executeWithOutput runs the application with args and returns what the
// command tree wrote to its output writer, including JSON error envelopes.
func executeWithOutput(t *testing.T, app *application, args []string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	app.rootCmd.SetOut(&out)
	err := app.execute(args)
	return out.String(), err
}

// runWithRootOutput runs the bare root command and returns its output.
func runWithRootOutput(t *testing.T, app *application) (string, error) {
	t.Helper()
	var out bytes.Buffer
	app.rootCmd.SetOut(&out)
	err := app.rootCmd.RunE(app.rootCmd, nil)
	return out.String(), err
}
