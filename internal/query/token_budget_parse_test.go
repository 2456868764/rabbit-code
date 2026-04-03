package query

import "testing"

func TestParseTokenBudget_shorthandStart(t *testing.T) {
	n, ok := ParseTokenBudget("+500k extra")
	if !ok || n != 500_000 {
		t.Fatalf("got %d ok=%v", n, ok)
	}
}

func TestParseTokenBudget_shorthandEnd(t *testing.T) {
	n, ok := ParseTokenBudget(`please use budget +2m.`)
	if !ok || n != 2_000_000 {
		t.Fatalf("got %d ok=%v", n, ok)
	}
}

func TestParseTokenBudget_verbose(t *testing.T) {
	n, ok := ParseTokenBudget("We should spend 3b tokens today")
	if !ok || n != 3_000_000_000 {
		t.Fatalf("got %d ok=%v", n, ok)
	}
}

func TestParseTokenBudget_none(t *testing.T) {
	if _, ok := ParseTokenBudget("no budget here"); ok {
		t.Fatal("expected false")
	}
}

func TestBudgetContinuationMessage_emDash(t *testing.T) {
	s := BudgetContinuationMessage(12, 1200, 10_000)
	if s == "" {
		t.Fatal("empty")
	}
}
