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
