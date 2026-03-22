//go:build !windows

package utils

import (
	"os"

	"github.com/google/renameio/v2"
)

// WriteFileAtomic writes data to a file atomically using write-to-temp + rename.
// On Unix, this delegates to renameio which handles fsync semantics correctly
// across different filesystem types.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	return renameio.WriteFile(path, data, perm)
}
