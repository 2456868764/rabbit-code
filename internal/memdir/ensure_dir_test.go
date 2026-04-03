package memdir

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEnsureMemoryDirExists(t *testing.T) {
	t.Parallel()
	if err := EnsureMemoryDirExists(""); err != nil {
		t.Fatalf("empty: %v", err)
	}
	base := t.TempDir()
	sub := filepath.Join(base, "nested", "mem")
	if err := EnsureMemoryDirExists(sub); err != nil {
		t.Fatal(err)
	}
	st, err := os.Stat(sub)
	if err != nil {
		t.Fatal(err)
	}
	if !st.IsDir() {
		t.Fatalf("want dir, got %v", st.Mode())
	}
}
