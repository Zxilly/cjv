package settings

import (
	"testing"

	"github.com/Zxilly/cjv/internal/cli/clitest"
)

// Some tests install toolchains; the harness keeps them off the real PATH.
func TestMain(m *testing.M) { clitest.Main(m) }
