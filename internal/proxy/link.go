package proxy

import (
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Zxilly/cjv/internal/config"
	sdktarget "github.com/Zxilly/cjv/internal/target"
	"github.com/Zxilly/cjv/internal/utils"
)

func CreateAllProxyLinks() error {
	binDir, err := config.BinDir()
	if err != nil {
		return err
	}

	cjvBinary := filepath.Join(binDir, CjvBinaryName())

	for _, tool := range AllProxyTools() {
		dst := filepath.Join(binDir, PlatformBinaryName(tool))
		if err := utils.CreateLink(cjvBinary, dst); err != nil {
			return err
		}
	}
	return nil
}

// CjvBinaryName returns the platform-appropriate cjv binary name.
func CjvBinaryName() string {
	return PlatformBinaryName("cjv")
}

// PlatformBinaryName returns the platform-appropriate binary name
// (appends .exe on Windows).
func PlatformBinaryName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

func PlatformBinaryNameForTuple(name, tuple string) (string, error) {
	parts, err := sdktarget.ParseTuple(tuple)
	if err != nil {
		return "", err
	}
	if strings.HasPrefix(parts.Host, "win32-") {
		return name + ".exe", nil
	}
	return name, nil
}
