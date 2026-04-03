package query

import (
	"strings"
	"time"
)

// BudgetTracker mirrors query/tokenBudget.ts BudgetTracker (H5.5).
type BudgetTracker struct {
	ContinuationCount    int
	LastDeltaTokens      int
	LastGlobalTurnTokens int
	StartedAtUnixMilli   int64
}

// NewBudgetTracker mirrors createBudgetTracker.
func NewBudgetTracker() BudgetTracker {
	return BudgetTracker{StartedAtUnixMilli: time.Now().UnixMilli()}
}

const (
	tokenBudgetCompletionThreshold = 0.9
	tokenBudgetDiminishingDelta    = 500
)

// BudgetAction mirrors TokenBudgetDecision branches.
type BudgetAction int

const (
	BudgetActionStop BudgetAction = iota
	BudgetActionContinue
)

// TokenBudgetCompletionEvent mirrors TS completionEvent when action stop with telemetry.
type TokenBudgetCompletionEvent struct {
	ContinuationCount  int
	Pct                int
	TurnTokens         int
	Budget             int
	DiminishingReturns bool
	DurationMs         int64
}

// TokenBudgetDecision mirrors query/tokenBudget.ts checkTokenBudget return (H5.5).
type TokenBudgetDecision struct {
	Action            BudgetAction
	NudgeMessage      string
	ContinuationCount int
	Pct               int
	TurnTokens        int
	Budget            int
	Completion        *TokenBudgetCompletionEvent
}

func budgetPctRounded(turnTokens, budget int) int {
	if budget <= 0 {
		return 0
	}
	return (turnTokens*100 + budget/2) / budget
}

// CheckTokenBudget mirrors query/tokenBudget.ts checkTokenBudget.
// agentID non-empty skips budgeting (forked / sub-agent). budget is the parsed per-turn output token cap; use <= 0 when unset.
func CheckTokenBudget(tracker *BudgetTracker, agentID string, budget int, globalTurnTokens int) TokenBudgetDecision {
	if tracker == nil {
		return TokenBudgetDecision{Action: BudgetActionStop, Completion: nil}
	}
	if strings.TrimSpace(agentID) != "" || budget <= 0 {
		return TokenBudgetDecision{Action: BudgetActionStop, Completion: nil}
	}

	turnTokens := globalTurnTokens
	pct := budgetPctRounded(turnTokens, budget)
	deltaSinceLast := turnTokens - tracker.LastGlobalTurnTokens

	isDiminishing := tracker.ContinuationCount >= 3 &&
		deltaSinceLast < tokenBudgetDiminishingDelta &&
		tracker.LastDeltaTokens < tokenBudgetDiminishingDelta

	if !isDiminishing && float64(turnTokens) < float64(budget)*tokenBudgetCompletionThreshold {
		tracker.ContinuationCount++
		tracker.LastDeltaTokens = deltaSinceLast
		tracker.LastGlobalTurnTokens = turnTokens
		return TokenBudgetDecision{
			Action:            BudgetActionContinue,
			NudgeMessage:      BudgetContinuationMessage(pct, turnTokens, budget),
			ContinuationCount: tracker.ContinuationCount,
			Pct:               pct,
			TurnTokens:        turnTokens,
			Budget:            budget,
		}
	}

	if isDiminishing || tracker.ContinuationCount > 0 {
		dur := time.Now().UnixMilli() - tracker.StartedAtUnixMilli
		return TokenBudgetDecision{
			Action: BudgetActionStop,
			Completion: &TokenBudgetCompletionEvent{
				ContinuationCount:  tracker.ContinuationCount,
				Pct:                pct,
				TurnTokens:         turnTokens,
				Budget:             budget,
				DiminishingReturns: isDiminishing,
				DurationMs:         dur,
			},
		}
	}

	return TokenBudgetDecision{Action: BudgetActionStop, Completion: nil}
}
