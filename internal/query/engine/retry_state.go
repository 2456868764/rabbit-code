package engine

import (
	"encoding/json"

	"github.com/2456868764/rabbit-code/internal/query"
	"github.com/2456868764/rabbit-code/internal/services/compact"
)

// resetLoopStateForRetryAttempt mirrors the prior partial preserve of *st before a second RunTurnLoop,
// but carries H6 bookkeeping (LoopContinue, overrides, auto-compact tracking) across attempts.
func resetLoopStateForRetryAttempt(st *query.LoopState) {
	p := *st
	*st = query.LoopState{
		MaxTurns:                      p.MaxTurns,
		CompactCount:                  p.CompactCount,
		MessagesJSON:                  append(json.RawMessage(nil), p.MessagesJSON...),
		ToolUseContext:                p.ToolUseContext,
		RecoveryAttempts:              p.RecoveryAttempts,
		RecoveryPhase:                 p.RecoveryPhase,
		LoopContinue:                  p.LoopContinue,
		AutoCompactTracking:           compact.CloneAutoCompactTracking(p.AutoCompactTracking),
		MaxOutputTokensRecoveryCount:  p.MaxOutputTokensRecoveryCount,
		HasAttemptedReactiveCompact:   p.HasAttemptedReactiveCompact,
		MaxOutputTokensOverrideActive: p.MaxOutputTokensOverrideActive,
		MaxOutputTokensOverride:       p.MaxOutputTokensOverride,
		PendingToolUseSummary:         p.PendingToolUseSummary,
		StopHookActive:                p.StopHookActive,
		SnipRemovalLog:                query.CloneSnipRemovalLog(p.SnipRemovalLog),
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
