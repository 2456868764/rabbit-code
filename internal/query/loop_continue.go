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
