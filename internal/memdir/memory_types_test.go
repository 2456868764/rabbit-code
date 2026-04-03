package memdir

import "testing"

func TestParseMemoryType(t *testing.T) {
	if ParseMemoryType("  project  ") != "project" {
		t.Fatal()
	}
	if ParseMemoryType("not-a-type") != "" {
		t.Fatal()
	}
	if ParseMemoryType("") != "" {
		t.Fatal()
	}
}
