package query

// Continue reasons mirror query.ts State.transition.reason (queryLoop continue sites).
const (
	ContinueReasonCollapseDrainRetry      = "collapse_drain_retry"
	ContinueReasonReactiveCompactRetry    = "reactive_compact_retry"
	ContinueReasonMaxOutputTokensEscalate = "max_output_tokens_escalate"
	ContinueReasonMaxOutputTokensRecovery = "max_output_tokens_recovery"
	ContinueReasonStopHookBlocking        = "stop_hook_blocking"
	// ContinueReasonStopHookPrevented mirrors query.ts terminal stop_hook_prevented (H6 headless).
	ContinueReasonStopHookPrevented = "stop_hook_prevented"
	ContinueReasonTokenBudgetContinuation = "token_budget_continuation"
	ContinueReasonNextTurn                = "next_turn"
	// ContinueReasonSubmitRecoverRetry is the engine-level second RunTurnLoop after RecoverStrategy (no 1:1 name in query.ts outer Submit).
	ContinueReasonSubmitRecoverRetry = "submit_recover_retry"
	// ContinueReasonAutoCompactExecuted records a successful post-loop auto compact executor run (headless bookkeeping).
	ContinueReasonAutoCompactExecuted = "auto_compact_executed"
	// ContinueReasonPromptCacheBreakTrimResend records strip cache_control + retry after stream cache break (H1).
	ContinueReasonPromptCacheBreakTrimResend = "prompt_cache_break_trim_resend"
	// ContinueReasonPromptCacheBreakCompactRetry records compact executor seed + retry after trim path failed (H1).
	ContinueReasonPromptCacheBreakCompactRetry = "prompt_cache_break_compact_retry"
)

// LoopContinue mirrors query.ts Continue (discriminated by Reason; optional payloads).
type LoopContinue struct {
	Reason    string
	Committed int // collapse_drain_retry: drained messages committed
	Attempt   int // max_output_tokens_recovery: 1-based attempt
}

// Empty reports no continue snapshot (query.ts transition undefined).
func (c LoopContinue) Empty() bool {
	return c.Reason == ""
}

// RecordLoopContinue sets st.LoopContinue (H6 / query.ts transition).
func RecordLoopContinue(st *LoopState, c LoopContinue) {
	if st == nil {
		return
	}
	st.LoopContinue = c
}

// ClearLoopContinue clears the continue snapshot.
func ClearLoopContinue(st *LoopState) {
	if st == nil {
		return
	}
	st.LoopContinue = LoopContinue{}
}

// ResetLoopStateFieldsForNextQueryIteration mirrors query.ts queryLoop's `next` State after tool results
// (before the recursive assistant call): maxOutputTokensRecoveryCount=0, hasAttemptedReactiveCompact=false,
// maxOutputTokensOverride cleared. See query.ts assignment to `next` before state = next.
func ResetLoopStateFieldsForNextQueryIteration(st *LoopState) {
	if st == nil {
		return
	}
	st.MaxOutputTokensRecoveryCount = 0
	st.HasAttemptedReactiveCompact = false
	st.MaxOutputTokensOverrideActive = false
	st.MaxOutputTokensOverride = 0
}
