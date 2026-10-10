package target

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// TestPlatformMatrixInSync guards against the shipped-platform set drifting
// between its single source of truth (SupportedHostPlatforms) and the parallel
// lists that must match it: the goreleaser build matrix (what actually gets
// built), the web landing page (what download links it offers), and the
// install-binary extraction script. A mismatch here means, e.g., the website
// could offer a binary goreleaser never built. The platforms are compared as
// "<goos>_<goarch>" tokens, which all four sources express in the same
// vocabulary.
func TestPlatformMatrixInSync(t *testing.T) {
	canonical := map[string]bool{}
	for _, p := range SupportedHostPlatforms() {
		canonical[p.GOOS+"_"+p.GOARCH] = true
	}
	if len(canonical) == 0 {
		t.Fatal("SupportedHostPlatforms returned no platforms")
	}

	root := repoRoot(t)
	sources := map[string]map[string]bool{
		".goreleaser.yml":                  goreleaserPlatforms(t, root),
		"web/src/generated/platforms.ts":   webPlatforms(t, root),
		"scripts/extract-init-binaries.sh": shellPlatforms(t, root),
	}

	for name, got := range sources {
		if !equalSets(canonical, got) {
			t.Errorf("%s platform set %v does not match SupportedHostPlatforms %v",
				name, sortedKeys(got), sortedKeys(canonical))
		}
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	// This test lives in internal/target, two levels below the repo root.
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	return root
}

func readRepoFile(t *testing.T, root, rel string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read %s: %v", rel, err)
	}
	return string(data)
}

// Both release variants must publish exactly the catalogued targets.
func goreleaserPlatforms(t *testing.T, root string) map[string]bool {
	content := readRepoFile(t, root, ".goreleaser.yml")
	matches := regexp.MustCompile(`targets:\s*\[([^\]]*)\]`).FindAllStringSubmatch(content, -1)
	if len(matches) != 2 {
		t.Fatalf("expected targets for both release variants, got %d", len(matches))
	}
	var common map[string]bool
	for _, match := range matches {
		set := map[string]bool{}
		for token := range strings.SplitSeq(match[1], ",") {
			parts := strings.Split(strings.TrimSpace(token), "_")
			if len(parts) < 2 {
				t.Fatalf("invalid release target %q", token)
			}
			set[parts[0]+"_"+parts[1]] = true
		}
		if common != nil && !equalSets(common, set) {
			t.Fatalf("release variants have different targets: %v vs %v", common, set)
		}
		common = set
	}
	return common
}

func webPlatforms(t *testing.T, root string) map[string]bool {
	content := readRepoFile(t, root, "web/src/generated/platforms.ts")
	set := map[string]bool{}
	re := regexp.MustCompile(`goos:\s*'([a-z0-9]+)',\s*goarch:\s*'([a-z0-9]+)'`)
	for _, m := range re.FindAllStringSubmatch(content, -1) {
		set[m[1]+"_"+m[2]] = true
	}
	if len(set) == 0 {
		t.Fatal("no generated platform entries parsed from platforms.ts")
	}
	return set
}

func shellPlatforms(t *testing.T, root string) map[string]bool {
	content := readRepoFile(t, root, "scripts/extract-init-binaries.sh")
	set := map[string]bool{}
	re := regexp.MustCompile(`"([a-z0-9]+)_([a-z0-9]+)\s+(?:zip|tar\.gz)"`)
	for _, m := range re.FindAllStringSubmatch(content, -1) {
		set[m[1]+"_"+m[2]] = true
	}
	if len(set) == 0 {
		t.Fatal("no PLATFORMS entries parsed from extract-init-binaries.sh")
	}
	return set
}

func equalSets(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
