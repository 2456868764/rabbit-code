package memdir

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindGitRoot_findsAncestor(t *testing.T) {
	top := t.TempDir()
	if err := os.MkdirAll(filepath.Join(top, "a", "b"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(top, ".git"), 0o700); err != nil {
		t.Fatal(err)
	}
	leaf := filepath.Join(top, "a", "b")
	got, ok := FindGitRoot(leaf)
	if !ok || got != top {
		t.Fatalf("got %q %v", got, ok)
	}
}

func TestFindGitRoot_none(t *testing.T) {
	dir := t.TempDir()
	_, ok := FindGitRoot(dir)
	if ok {
		t.Fatal("expected no git root")
	}
}
