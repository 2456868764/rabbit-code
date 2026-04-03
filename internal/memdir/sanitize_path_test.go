package memdir

import (
	"strings"
	"testing"
)

func TestSanitizePath_ascii(t *testing.T) {
	got := SanitizePath("/Users/foo/my-project")
	want := "-Users-foo-my-project"
	if got != want {
		t.Fatalf("%q", got)
	}
}

func TestSanitizePath_truncatesWithHash(t *testing.T) {
	long := strings.Repeat("a", MaxSanitizedLength+30)
	got := SanitizePath(long)
	if len(got) <= MaxSanitizedLength {
		t.Fatalf("expected hash suffix extension, got len %d", len(got))
	}
	if !strings.Contains(got, "-") {
		t.Fatal(got)
	}
}
