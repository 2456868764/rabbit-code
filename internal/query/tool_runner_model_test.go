package query

import (
	"testing"
)

func TestResolveMainLoopModel(t *testing.T) {
	t.Setenv("ANTHROPIC_MODEL", "")
	if got := ResolveMainLoopModel("  m1 "); got != "m1" {
		t.Fatal(got)
	}
	t.Setenv("ANTHROPIC_MODEL", "from-env")
	if got := ResolveMainLoopModel(""); got != "from-env" {
		t.Fatal(got)
	}
	t.Setenv("ANTHROPIC_MODEL", "")
	if got := ResolveMainLoopModel(""); got != "claude-3-5-haiku-20241022" {
		t.Fatal(got)
	}
}
