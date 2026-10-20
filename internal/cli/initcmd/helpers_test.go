package initcmd

import (
	"testing"

	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/cli/clitest"
)

func TestMain(m *testing.M) { clitest.Main(m) }

// newTestCommands returns init with its flag defaults applied.
func newTestCommands() *commands {
	c := &commands{ctx: clictx.New("dev", "")}
	c.command()
	return c
}
