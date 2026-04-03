package engine

import "github.com/2456868764/rabbit-code/internal/query"

// resetLoopStateForRetryAttempt mirrors the prior partial preserve of *st before a second RunTurnLoop,
// but carries H6 bookkeeping (LoopContinue, overrides, auto-compact tracking) across attempts.
func resetLoopStateForRetryAttempt(st *query.LoopState) {
	p := *st
	*st = query.LoopState{
		MaxTurns:                      p.MaxTurns,
		CompactCount:                  p.CompactCount,
		RecoveryAttempts:              p.RecoveryAttempts,
		RecoveryPhase:                 p.RecoveryPhase,
		LoopContinue:                  p.LoopContinue,
		AutoCompactTracking:           query.CloneAutoCompactTracking(p.AutoCompactTracking),
		MaxOutputTokensRecoveryCount:  p.MaxOutputTokensRecoveryCount,
		HasAttemptedReactiveCompact:   p.HasAttemptedReactiveCompact,
		MaxOutputTokensOverrideActive: p.MaxOutputTokensOverrideActive,
		MaxOutputTokensOverride:       p.MaxOutputTokensOverride,
		PendingToolUseSummary:         p.PendingToolUseSummary,
		StopHookActive:                p.StopHookActive,
	}
	st.RecoveryPhase = query.RecoveryRetriedOnce
}

// PrepareLoopStateForStopHookBlockingContinuation mirrors query.ts carry-over after stop_hook_blocking (H6).
func PrepareLoopStateForStopHookBlockingContinuation(st *query.LoopState) {
	if st == nil {
		return
	}
	st.MaxOutputTokensRecoveryCount = 0
	st.MaxOutputTokensOverrideActive = false
	st.MaxOutputTokensOverride = 0
	st.PendingToolUseSummary = false
	st.StopHookActive = true
}

// PrepareLoopStateForTokenBudgetContinuation mirrors query.ts carry-over after token_budget_continuation (H6).
func PrepareLoopStateForTokenBudgetContinuation(st *query.LoopState) {
	if st == nil {
		return
	}
	st.MaxOutputTokensRecoveryCount = 0
	st.HasAttemptedReactiveCompact = false
	st.MaxOutputTokensOverrideActive = false
	st.MaxOutputTokensOverride = 0
	st.PendingToolUseSummary = false
	st.StopHookActive = false
}
