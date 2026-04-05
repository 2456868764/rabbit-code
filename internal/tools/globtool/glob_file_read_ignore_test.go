package globtool

import (
	"path/filepath"
	"testing"
)

func TestNormalizeFileReadDenyPatternsToSearchDir_nullRoot(t *testing.T) {
	got := NormalizeFileReadDenyPatternsToSearchDir(map[string][]string{
		"": {"**/secret/**", "!already"},
	}, "/tmp/proj")
	seen := map[string]bool{}
	for _, g := range got {
		seen[g] = true
	}
	if !seen["!**/secret/**"] || !seen["!already"] {
		t.Fatalf("%v", got)
	}
}

func TestNormalizeFileReadDenyPatternsToSearchDir_sameRoot(t *testing.T) {
	root := filepath.Clean(t.TempDir())
	got := NormalizeFileReadDenyPatternsToSearchDir(map[string][]string{
		root: {"old/**"},
	}, root)
	if len(got) != 1 || got[0] != "!/old/**" {
		t.Fatalf("%v", got)
	}
}

func TestNormalizeFileReadDenyPatternsToSearchDir_nestedRoot(t *testing.T) {
	proj := filepath.Clean(t.TempDir())
	sub := filepath.Join(proj, "pkg")
	got := NormalizeFileReadDenyPatternsToSearchDir(map[string][]string{
		sub: {"*.env"},
	}, proj)
	if len(got) != 1 || got[0] != "!/pkg/*.env" {
		t.Fatalf("%v", got)
	}
}
