package memdir

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

func TestSessionFragmentsFromPathsWithAttachmentHeadersAt(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m.md")
	_ = os.WriteFile(p, []byte("  body  "), 0o600)
	now := time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC)
	frags, raw, err := SessionFragmentsFromPathsWithAttachmentHeadersAt([]string{p}, now)
	if err != nil {
		t.Fatal(err)
	}
	if len(frags) != 1 {
		t.Fatalf("%#v", frags)
	}
	if !strings.Contains(frags[0], "body") || !strings.Contains(frags[0], p) {
		t.Fatalf("%q", frags[0])
	}
	if raw != len(frags[0]) {
		t.Fatalf("raw %d len frag %d", raw, len(frags[0]))
	}
}
