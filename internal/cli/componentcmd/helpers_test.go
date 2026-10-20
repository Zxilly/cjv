package componentcmd

import (
	"testing"

	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/cli/clitest"
)

func TestMain(m *testing.M) { clitest.Main(m) }

func newTestCommands() *commands {
	return newCommands(clictx.New("dev", ""))
}

// execute runs args through a fresh root holding the component command.
func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()
	return clitest.Execute(t, clitest.Root(Register), args...)
}
