package memdir

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindRelevantMemoryPaths(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "alpha-notes.md"), []byte("discussion about bananas"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "other.txt"), []byte("bananas"), 0o600)
	paths, err := FindRelevantMemoryPaths("banana bread", dir, 3)
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || filepath.Base(paths[0]) != "alpha-notes.md" {
		t.Fatalf("%#v", paths)
	}
}

func TestFindRelevantMemoryPaths_emptyQuery(t *testing.T) {
	p, err := FindRelevantMemoryPaths("   ", t.TempDir(), 5)
	if err != nil || len(p) != 0 {
		t.Fatalf("%v %#v", err, p)
	}
}
