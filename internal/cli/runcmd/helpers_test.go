package runcmd

import (
	"os"
	"testing"

	"github.com/Zxilly/cjv/internal/cli/clictx"
	"github.com/Zxilly/cjv/internal/cli/clitest"
)

func TestMain(m *testing.M) {
	if os.Getenv("CJV_TEST_INVOCATION_REPORT") != "" {
		// This child only reports arguments and environment. Its parent already
		// holds the Windows PATH guard, so acquiring it again would deadlock.
		os.Exit(m.Run())
	}
	clitest.Main(m)
}

func newTestCommands() *commands {
	return &commands{ctx: clictx.New("dev", "")}
}

// execute runs args through a fresh root holding this package's commands.
func execute(t *testing.T, args ...string) (string, error) {
	t.Helper()
	return clitest.Execute(t, clitest.Root(Register), args...)
}
