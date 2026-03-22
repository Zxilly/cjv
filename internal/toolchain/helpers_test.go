package toolchain

import (
	"os"
	"testing"

	"github.com/Zxilly/cjv/internal/config"
)

func TestMain(m *testing.M) {
	config.ResetDefaultSettingsFileCache()
	os.Exit(m.Run())
}
