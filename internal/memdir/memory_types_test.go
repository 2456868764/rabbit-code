package memdir

import (
	"strings"
	"testing"
)

func TestWhenToAccessSection_includesMemoryDriftCaveat(t *testing.T) {
	lines := WhenToAccessSection()
	var joined string
	for _, ln := range lines {
		joined += ln + "\n"
	}
	if !strings.Contains(joined, "Memory records can become stale over time") {
		t.Fatal("drift caveat missing from when-to-access section")
	}
}

func TestTypesSectionIndividual_nonEmpty(t *testing.T) {
	if len(TypesSectionIndividual()) < 10 {
		t.Fatalf("unexpectedly short: %d", len(TypesSectionIndividual()))
	}
}

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
