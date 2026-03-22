package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseToolchainFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "cangjie-sdk.toml")
	os.WriteFile(path, []byte(`[toolchain]
channel = "lts-1.0.5"
`), 0o644)

	tc, err := ParseToolchainFile(path)
	require.NoError(t, err)
	assert.Equal(t, "lts-1.0.5", tc.Toolchain.Channel)
}

func TestFindToolchainFile(t *testing.T) {
	// Create nested directory structure
	root := t.TempDir()
	sub := filepath.Join(root, "a", "b", "c")
	os.MkdirAll(sub, 0o755)

	// Place cangjie-sdk.toml in root/a/
	os.WriteFile(filepath.Join(root, "a", "cangjie-sdk.toml"), []byte(`[toolchain]
channel = "sts"
`), 0o644)

	// Walk up from root/a/b/c
	path, err := FindToolchainFile(sub)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "a", "cangjie-sdk.toml"), path)
}

func TestFindToolchainFileNotFound(t *testing.T) {
	tmp := t.TempDir()
	path, err := FindToolchainFile(tmp)
	require.NoError(t, err)
	assert.Equal(t, "", path)
}

// --- Tests merged from toolchain_file_edge_test.go ---

func TestParseToolchainFile_MalformedToml(t *testing.T) {
	// Users might hand-edit cangjie-sdk.toml and introduce syntax errors.
	// The error must be reported, not silently ignored.
	path := filepath.Join(t.TempDir(), "cangjie-sdk.toml")
	require.NoError(t, os.WriteFile(path, []byte("[toolchain\nchannel"), 0o644))

	_, err := ParseToolchainFile(path)
	assert.Error(t, err, "malformed TOML should be reported as error")
}

func TestParseToolchainFile_EmptyChannelIsAccepted(t *testing.T) {
	// An empty channel field is valid TOML. ParseToolchainFile should
	// parse it — validation of the empty channel is the caller's job
	// (ResolveToolchain returns a specific error for this).
	path := filepath.Join(t.TempDir(), "cangjie-sdk.toml")
	content := "[toolchain]\nchannel = \"\"\n"
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	tc, err := ParseToolchainFile(path)
	require.NoError(t, err)
	assert.Empty(t, tc.Toolchain.Channel)
}

func TestParseToolchainFile_EmptyFile(t *testing.T) {
	// Empty file is valid TOML (no sections).
	// Toolchain.Channel will be zero value (empty string).
	path := filepath.Join(t.TempDir(), "cangjie-sdk.toml")
	require.NoError(t, os.WriteFile(path, []byte(""), 0o644))

	tc, err := ParseToolchainFile(path)
	require.NoError(t, err)
	assert.Empty(t, tc.Toolchain.Channel)
}

func TestParseToolchainFile_ValidWithAllFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cangjie-sdk.toml")
	content := `[toolchain]
channel = "lts"
components = ["std", "gui"]
targets = ["win32-x64"]
profile = "full"
`
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))

	tc, err := ParseToolchainFile(path)
	require.NoError(t, err)
	assert.Equal(t, "lts", tc.Toolchain.Channel)
	assert.Equal(t, []string{"std", "gui"}, tc.Toolchain.Components)
	assert.Equal(t, []string{"win32-x64"}, tc.Toolchain.Targets)
	assert.Equal(t, "full", tc.Toolchain.Profile)
}

func TestParseToolchainFile_NonExistentFile(t *testing.T) {
	_, err := ParseToolchainFile(filepath.Join(t.TempDir(), "nonexistent.toml"))
	assert.Error(t, err)
}

// --- Tests merged from find_toolchain_file_test.go ---

func TestFindToolchainFile_InCurrentDir(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cangjie-sdk.toml"),
		[]byte("[toolchain]\nchannel = \"lts\"\n"), 0o644))

	path, err := FindToolchainFile(dir)
	require.NoError(t, err)
	assert.NotEmpty(t, path)
	assert.Contains(t, path, "cangjie-sdk.toml")
}

func TestFindToolchainFile_NotFound(t *testing.T) {
	// Temporary directory has no cangjie-sdk.toml
	dir := t.TempDir()
	path, err := FindToolchainFile(dir)
	assert.NoError(t, err) // not finding is not an error
	assert.Empty(t, path)
}

func TestFindToolchainFile_InParentDir(t *testing.T) {
	parent := t.TempDir()
	child := filepath.Join(parent, "subdir")
	require.NoError(t, os.MkdirAll(child, 0o755))

	require.NoError(t, os.WriteFile(filepath.Join(parent, "cangjie-sdk.toml"),
		[]byte("[toolchain]\nchannel = \"sts\"\n"), 0o644))

	path, err := FindToolchainFile(child)
	require.NoError(t, err)
	assert.Contains(t, path, "cangjie-sdk.toml")
}
