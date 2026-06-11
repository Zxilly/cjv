package target

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
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

// goreleaserPlatforms derives the build matrix from the goos/goarch arrays
// minus the ignore pairs. .goreleaser.yml defines multiple build blocks (cjv
// and cjv-mirror), each with its own goos/goarch; this parses ALL of them and
// requires every block's matrix to be identical, so a divergence in any build
// (not just the first) is caught.
func goreleaserPlatforms(t *testing.T, root string) map[string]bool {
	content := readRepoFile(t, root, ".goreleaser.yml")

	listAll := func(key string) [][]string {
		matches := regexp.MustCompile(key+`:\s*\[([^\]]*)\]`).FindAllStringSubmatch(content, -1)
		if len(matches) == 0 {
			t.Fatalf("could not find any %s array in .goreleaser.yml", key)
		}
		var lists [][]string
		for _, m := range matches {
			var out []string
			for tok := range strings.SplitSeq(m[1], ",") {
				if tok = strings.TrimSpace(tok); tok != "" {
					out = append(out, tok)
				}
			}
			lists = append(lists, out)
		}
		return lists
	}

	// Every build block must declare the same matrix so cjv and cjv-mirror
	// cannot drift apart unnoticed.
	common := func(key string) []string {
		lists := listAll(key)
		for i := 1; i < len(lists); i++ {
			if !slices.Equal(lists[i], lists[0]) {
				t.Fatalf("%s arrays differ between goreleaser build blocks: %v vs %v", key, lists[0], lists[i])
			}
		}
		return lists[0]
	}
	gooses := common("goos")
	goarches := common("goarch")

	// Ignore pairs are collected across all blocks; the matrix-equality check
	// above already guarantees the blocks agree, so a merged ignore set is sound.
	ignored := map[string]bool{}
	ignoreRE := regexp.MustCompile(`-\s*goos:\s*(\S+)\s*\n\s*goarch:\s*(\S+)`)
	for _, m := range ignoreRE.FindAllStringSubmatch(content, -1) {
		ignored[m[1]+"_"+m[2]] = true
	}

	set := map[string]bool{}
	for _, os := range gooses {
		for _, arch := range goarches {
			key := os + "_" + arch
			if !ignored[key] {
				set[key] = true
			}
		}
	}
	return set
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
