package query

import (
	"strings"
	"testing"
)

func TestApplyPhase5UserTextHints(t *testing.T) {
	out := ApplyPhase5UserTextHints("hello", Phase5UserTextFlags{Ultrathink: true})
	if !strings.HasPrefix(out, "[ULTRATHINK:") || !strings.Contains(out, "hello") {
		t.Fatalf("%q", out)
	}
	out2 := ApplyPhase5UserTextHints("x", Phase5UserTextFlags{ContextCollapse: true, Ultraplan: true})
	for _, sub := range []string{"x", "CONTEXT_COLLAPSE", "ULTRAPLAN"} {
		if !strings.Contains(out2, sub) {
			t.Fatalf("missing %q in %q", sub, out2)
		}
	}
	out3 := ApplyPhase5UserTextHints("z", Phase5UserTextFlags{SessionRestore: true})
	if !strings.Contains(out3, "SESSION_RESTORE") {
		t.Fatalf("%q", out3)
	}
}

func TestFormatPhase5HeadlessModeTags_order(t *testing.T) {
	got := FormatPhase5HeadlessModeTags(Phase5UserTextFlags{
		Ultraplan: true, Ultrathink: true, ContextCollapse: true,
	})
	if got != "context_collapse,ultrathink,ultraplan" {
		t.Fatalf("%q", got)
	}
}
