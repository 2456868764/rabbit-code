package bashtool

import "testing"

func TestApplySedSubstitution_global(t *testing.T) {
	info := ParseSedEditCommand(`sed -i '' 's/a/b/g' f.txt`)
	if info == nil {
		t.Fatal("parse")
	}
	got := ApplySedSubstitution("a a", info)
	if got != "b b" {
		t.Fatalf("got %q", got)
	}
}

func TestApplySedSubstitution_firstOnly(t *testing.T) {
	info := ParseSedEditCommand(`sed -i '' 's/foo/bar/' f.txt`)
	if info == nil {
		t.Fatal("parse")
	}
	got := ApplySedSubstitution("foo foo", info)
	if got != "bar foo" {
		t.Fatalf("got %q", got)
	}
}

func TestApplySedSubstitution_ampersand(t *testing.T) {
	info := ParseSedEditCommand(`sed -i '' 's/hi/X&Y/g' f.txt`)
	if info == nil {
		t.Fatal("parse")
	}
	got := ApplySedSubstitution("hi", info)
	if got != "XhiY" {
		t.Fatalf("got %q", got)
	}
}
