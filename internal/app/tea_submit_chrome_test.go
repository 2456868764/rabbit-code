package app

import (
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/query/engine"
)

func TestSubmitChromeTeaModel_InitAndUpdate(t *testing.T) {
	ch := make(chan engine.EngineEvent, 2)
	ch <- engine.EngineEvent{
		Kind:         engine.EventKindSubmitTokenBudgetSnapshot,
		PhaseAuxInt:  99,
		PhaseAuxInt2: 0,
		PhaseDetail:  "bytes4",
	}
	ch <- engine.EngineEvent{Kind: engine.EventKindMemdirInject, MemdirFragmentCount: 1}

	m := NewSubmitChromeTeaModel(ch)
	cmd := m.Init()
	if cmd == nil {
		t.Fatal("expected listen cmd")
	}
	msg := cmd()
	evm, ok := msg.(EngineEventMsg)
	if !ok {
		t.Fatalf("got %T", msg)
	}
	m2, cmd2 := m.Update(evm)
	m = m2.(*SubmitChromeTeaModel)
	if cmd2 == nil {
		t.Fatal("expected chained listen")
	}
	msg2 := cmd2()
	evm2, ok := msg2.(EngineEventMsg)
	if !ok {
		t.Fatalf("second %T", msg2)
	}
	m3, _ := m.Update(evm2)
	m = m3.(*SubmitChromeTeaModel)

	v := m.View()
	if v.Content == "" {
		t.Fatal("expected styled view")
	}
	// lipgloss adds ansi; still expect raw substrings in content
	if !strings.Contains(v.Content, "99") || !strings.Contains(v.Content, "memdir") {
		t.Fatalf("view %q", v.Content)
	}
}
