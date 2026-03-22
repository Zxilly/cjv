package config

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	ResetDefaultSettingsFileCache()
	os.Exit(m.Run())
}
