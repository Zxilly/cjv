package env

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Zxilly/cjv/internal/utils"
)

const (
	markerStart = "# cjv (managed by cjv, do not edit)"
	markerEnd   = "# cjv end"
)

// filePermOrDefault returns the file's current permissions, or fallback for new files.
func filePermOrDefault(path string, fallback os.FileMode) os.FileMode {
	info, err := os.Stat(path)
	if err != nil {
		return fallback
	}
	return info.Mode().Perm()
}

// addBlockToShellConfig appends a marker-delimited block to a shell config
// file. It is idempotent: if the marker is already present, it does nothing.
func addBlockToShellConfig(configPath, block string) error {
	content, err := os.ReadFile(configPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}

	str := string(content)

	if strings.Contains(str, markerStart) {
		return nil
	}

	perm := filePermOrDefault(configPath, 0o644)
	return utils.WriteFileAtomic(configPath, []byte(str+block), perm)
}

// AddPathToShellConfig adds a PATH export block to a shell config file
// (e.g., .bashrc, .zshrc) using marker comments. Idempotent.
func AddPathToShellConfig(configPath string, binDir string) error {
	block := fmt.Sprintf("\n%s\nexport PATH='%s':\"$PATH\"\n%s\n", markerStart, strings.ReplaceAll(binDir, "'", "'\\''"), markerEnd)
	return addBlockToShellConfig(configPath, block)
}

// AddPathToFishConfig adds a fish_add_path block to fish config.
func AddPathToFishConfig(configPath string, binDir string) error {
	// Fish uses \' to escape single quotes inside single-quoted strings (since fish 3.1),
	// unlike POSIX shells which use the '\'' concatenation trick.
	block := fmt.Sprintf("\n%s\nfish_add_path -g '%s'\n%s\n", markerStart, strings.ReplaceAll(binDir, "'", "\\'"), markerEnd)
	return addBlockToShellConfig(configPath, block)
}

// ShellConfigPaths returns the list of shell config files and the fish config file.
// The first return value is the list of POSIX shell configs (.bashrc, .zshrc).
// The second return value is the fish config path (empty if fish config dir doesn't exist).
func ShellConfigPaths() (posix []string, fish string) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, ""
	}
	for _, rc := range []string{".profile", ".bashrc", ".zshrc", ".zprofile"} {
		posix = append(posix, filepath.Join(homeDir, rc))
	}
	fishDir := filepath.Join(homeDir, ".config", "fish")
	if _, err := os.Stat(fishDir); err == nil {
		fish = filepath.Join(fishDir, "config.fish")
	}
	return posix, fish
}

// RemovePathFromShellConfig removes the cjv marker block from a shell config file.
func RemovePathFromShellConfig(configPath string) error {
	content, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	str := string(content)
	changed := false
	for {
		startIdx := strings.Index(str, markerStart)
		if startIdx == -1 {
			break
		}

		// Search for markerEnd AFTER markerStart to avoid incorrect ordering
		endIdx := strings.Index(str[startIdx:], markerEnd)
		if endIdx == -1 {
			return fmt.Errorf("cjv block in %s has start marker but is missing end marker; please remove it manually", configPath)
		}
		endIdx = startIdx + endIdx + len(markerEnd)

		// Also remove surrounding newlines
		if startIdx > 0 && str[startIdx-1] == '\n' {
			startIdx--
		}
		if endIdx < len(str) && str[endIdx] == '\n' {
			endIdx++
		}

		str = str[:startIdx] + str[endIdx:]
		changed = true
	}

	if !changed {
		return nil
	}

	perm := filePermOrDefault(configPath, 0o644)
	return utils.WriteFileAtomic(configPath, []byte(str), perm)
}
