package query

import "testing"

func TestCheckTokenBudget_skipsWithAgentID(t *testing.T) {
	tr := NewBudgetTracker()
	d := CheckTokenBudget(&tr, "sub-1", 1000, 0)
	if d.Action != BudgetActionStop || d.Completion != nil {
		t.Fatalf("%+v", d)
	}
}

func TestCheckTokenBudget_skipsZeroBudget(t *testing.T) {
	tr := NewBudgetTracker()
	d := CheckTokenBudget(&tr, "", 0, 0)
	if d.Action != BudgetActionStop || d.Completion != nil {
		t.Fatalf("%+v", d)
	}
}

func TestCheckTokenBudget_continueUnderThreshold(t *testing.T) {
	tr := NewBudgetTracker()
	d := CheckTokenBudget(&tr, "", 10_000, 100)
	if d.Action != BudgetActionContinue || d.NudgeMessage == "" {
		t.Fatalf("%+v", d)
	}
	if tr.ContinuationCount != 1 {
		t.Fatalf("tracker continuation %d", tr.ContinuationCount)
	}
}

func TestCheckTokenBudget_diminishingReturns(t *testing.T) {
	tr := BudgetTracker{
		ContinuationCount:    3,
		LastDeltaTokens:      10,
		LastGlobalTurnTokens: 5000,
		StartedAtUnixMilli:   1,
	}
	// No progress: delta 0, last delta small -> diminishing; should stop with completion.
	d := CheckTokenBudget(&tr, "", 100_000, 5000)
	if d.Action != BudgetActionStop || d.Completion == nil || !d.Completion.DiminishingReturns {
		t.Fatalf("%+v", d)
	}
}
