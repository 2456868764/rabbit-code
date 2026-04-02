package memdir

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSessionFragmentsFromPaths(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.txt")
	b := filepath.Join(dir, "b.txt")
	_ = os.WriteFile(a, []byte("  hello  \n"), 0o600)
	_ = os.WriteFile(b, []byte("world"), 0o600)
	frags, raw, err := SessionFragmentsFromPaths([]string{a, b})
	if err != nil {
		t.Fatal(err)
	}
	if raw != len([]byte("  hello  \n"))+len([]byte("world")) {
		t.Fatalf("raw bytes %d", raw)
	}
	if len(frags) != 2 || frags[0] != "hello" || frags[1] != "world" {
		t.Fatalf("%#v", frags)
	}
}

func TestSessionFragmentsFromPaths_missing(t *testing.T) {
	_, _, err := SessionFragmentsFromPaths([]string{"/nonexistent/memdir.txt"})
	if err == nil {
		t.Fatal("expected error")
	}
}
