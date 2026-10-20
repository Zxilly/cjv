//go:build windows

package lifecycle

import "os"

// Windows directory links and junctions retain the existing transaction.
// Their deletion semantics differ from a Unix file-only unlinkat operation.
func linkedToolchainRemover(os.FileInfo) func(*os.Root, string) error { return nil }
