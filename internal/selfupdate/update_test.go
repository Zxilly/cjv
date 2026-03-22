package selfupdate

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Zxilly/cjv/internal/proxy"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Tests for Update — self-update mechanism.

func TestUpdate_DevVersionIsAlwaysUpToDate(t *testing.T) {
	// When built from source (version="dev"), there's no release to update to.
	// Update should return early as "already up to date".
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	err := Update(context.Background(), "https://github.com/Zxilly/cjv", "dev")
	assert.NoError(t, err)
}

func TestUpdate_WithManagedBinaryAndDevVersion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CJV_HOME", home)

	binDir := filepath.Join(home, "bin")
	require.NoError(t, os.MkdirAll(binDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(binDir, proxy.CjvBinaryName()), []byte("stub"), 0o755))

	err := Update(context.Background(), "https://github.com/Zxilly/cjv", "dev")
	assert.NoError(t, err, "dev version should short-circuit as up to date")
}

// --- Tests merged from slug_test.go ---

// Tests for extractSlug -- normalizes a mirror URL into an "owner/repo"
// slug for the GitHub Releases API. Users configure the mirror via
// settings; it must work with both full URLs and bare slugs.

func TestExtractSlug_FullGitHubURL(t *testing.T) {
	// User pastes a full GitHub URL from their browser.
	slug := extractSlug("https://github.com/Zxilly/cjv/releases")
	assert.Equal(t, "Zxilly/cjv", slug)
}

func TestExtractSlug_URLWithoutTrailingPath(t *testing.T) {
	slug := extractSlug("https://github.com/Zxilly/cjv")
	assert.Equal(t, "Zxilly/cjv", slug)
}

func TestExtractSlug_AlreadyASlug(t *testing.T) {
	// User writes "owner/repo" directly in config -- should pass through.
	slug := extractSlug("Zxilly/cjv")
	assert.Equal(t, "Zxilly/cjv", slug)
}

func TestExtractSlug_URLWithDeepPath(t *testing.T) {
	// URL with extra path segments beyond owner/repo.
	slug := extractSlug("https://github.com/golang/go/tree/master/src")
	assert.Equal(t, "golang/go", slug)
}

func TestExtractSlug_BareStringPassthrough(t *testing.T) {
	// Input without "/" or "://" is returned as-is.
	slug := extractSlug("foobar")
	assert.Equal(t, "foobar", slug)
}

// --- Tests merged from update_slug_test.go ---

// Additional edge-case tests for extractSlug.

func TestExtractSlug_TrailingSlash(t *testing.T) {
	slug := extractSlug("https://github.com/Zxilly/cjv/")
	assert.Equal(t, "Zxilly/cjv", slug)
}

func TestExtractSlug_SingleSegment(t *testing.T) {
	// URL with only one path segment — should return original
	slug := extractSlug("https://github.com/Zxilly")
	assert.Equal(t, "https://github.com/Zxilly", slug)
}
