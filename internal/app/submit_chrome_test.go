package app

import (
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/query/engine"
)

func TestSubmitChromeState_FormatSubmitChromeLine(t *testing.T) {
	var s SubmitChromeState
	if s.FormatSubmitChromeLine() != "" {
		t.Fatal("expected empty")
	}
	s.ApplyEngineEvent(engine.EngineEvent{
		Kind:         engine.EventKindSubmitTokenBudgetSnapshot,
		PhaseAuxInt:  1200,
		PhaseAuxInt2: 400,
		PhaseDetail:  "structured",
	})
	line := s.FormatSubmitChromeLine()
	if line == "" || !strings.Contains(line, "1200") || !strings.Contains(line, "400") || !strings.Contains(line, "structured") {
		t.Fatalf("line %q", line)
	}
	s.ApplyEngineEvent(engine.EngineEvent{Kind: engine.EventKindMemdirInject, MemdirFragmentCount: 2})
	line = s.FormatSubmitChromeLine()
	if !strings.Contains(line, "memdir") || !strings.Contains(line, "+2") {
		t.Fatalf("line %q", line)
	}
}
