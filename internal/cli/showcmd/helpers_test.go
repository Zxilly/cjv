package showcmd

import (
	"testing"

	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/cli/clitest"
)

func TestMain(m *testing.M) { clitest.Main(m) }

func newTestCommands() *commands {
	return newCommands(clictx.New("dev", ""))
}
