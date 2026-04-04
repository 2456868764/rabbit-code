package query

import (
	"encoding/json"
	"fmt"

	"github.com/2456868764/rabbit-code/internal/services/compact"
)

// RecoveryPhase is a coarse recovery lane for LoopState (P5.1.1; full query.ts parity in later phases).
type RecoveryPhase uint8

const (
	RecoveryNone RecoveryPhase = iota
	RecoveryPendingCompact
	RecoveryRetriedOnce
)

// ToolUseContextMirror holds a headless subset of query.ts ToolUseContext (H6).
// Full TS type includes AppState, MCP, hooks, etc.; PARITY follow-on extends this struct.
type ToolUseContextMirror struct {
	AgentID        string
	MainLoopModel  string
	NonInteractive bool
	QueryChainID   string
	QueryDepth     int
	// SessionID optional analytics / session key (query.ts session-adjacent wiring).
	SessionID string
	// Debug mirrors toolUseContext.options.debug when set via engine.Config.
	Debug bool
	// AbortSignalAborted is true after a context.Canceled assistant/tool path in this RunTurnLoop invocation.
	AbortSignalAborted bool
	// QuerySource mirrors query.ts QuerySource when the loop runs inside a forked agent (headless subset).
	QuerySource string
}

// LoopState tracks cross-iteration query loop metadata aligned with query.ts State (H6).
type LoopState struct {
	TurnCount    int
	PendingTools int
	InCompact    bool
	// MaxTurns if > 0 caps assistant turns (enforcement in loop later).
	MaxTurns int
	// CompactCount increments on TranStartCompact (P5.1.1 / P5.2.1 bookkeeping).
	CompactCount int
	// MessagesJSON mirrors query.ts state.messages in API JSON array form; updated on each transcript mutation in RunTurnLoop.
	MessagesJSON json.RawMessage
	// ToolUseContext mirrors a subset of query.ts toolUseContext (see ToolUseContextMirror).
	ToolUseContext ToolUseContextMirror
	// LoopContinue mirrors query.ts transition (why the previous iteration continued).
	LoopContinue LoopContinue
	// AutoCompactTracking mirrors query.ts autoCompactTracking (nil = undefined).
	AutoCompactTracking *compact.AutoCompactTracking
	// MaxOutputTokensRecoveryCount mirrors query.ts maxOutputTokensRecoveryCount.
	MaxOutputTokensRecoveryCount int
	// HasAttemptedReactiveCompact mirrors query.ts hasAttemptedReactiveCompact.
	HasAttemptedReactiveCompact bool
	// MaxOutputTokensOverride mirrors query.ts maxOutputTokensOverride when OverrideActive is set.
	MaxOutputTokensOverrideActive bool
	MaxOutputTokensOverride       int
	// PendingToolUseSummary mirrors query.ts pendingToolUseSummary presence (Promise → bool for headless).
	PendingToolUseSummary bool
	// StopHookActive mirrors query.ts stopHookActive (TS undefined → false in Go).
	StopHookActive bool
	// Recovery / stream metadata (P5.1.1).
	RecoveryPhase    RecoveryPhase
	RecoveryAttempts int
	LastStopReason   string
	HadStreamError   bool
	// LastAPIErrorKind is the anthropic.APIError kind string after a failed assistant call (P5.1.3).
	LastAPIErrorKind string
	// SnipRemovalLog records automatic prefix trims (H7); marshal with query.MarshalSnipRemovalLogJSON for session sidecars.
	SnipRemovalLog []SnipRemovalEntry
}

// SetMessagesJSON replaces MessagesJSON with a copy of msgs (query.ts state.messages mirror, H6).
func (st *LoopState) SetMessagesJSON(msgs json.RawMessage) {
	if st == nil {
		return
	}
	if len(msgs) == 0 {
		st.MessagesJSON = nil
		return
	}
	st.MessagesJSON = json.RawMessage(append([]byte(nil), msgs...))
}

// Transition names logical query loop edges (table-driven tests, AC5-1).
type Transition string

const (
	TranReceiveAssistant Transition = "receive_assistant"
	TranScheduleTools    Transition = "schedule_tools"
	TranToolCallsDone    Transition = "tool_calls_done"
	TranStartCompact     Transition = "start_compact"
	TranFinishCompact    Transition = "finish_compact"
)

// ApplyTransition returns the next state (pure, no I/O).
// MessagesJSON and ToolUseContext are not mutated by transitions (they mirror query.ts state carried outside transition edges).
func ApplyTransition(s LoopState, t Transition) LoopState {
	out := s
	switch t {
	case TranReceiveAssistant:
		out.TurnCount++
	case TranScheduleTools:
		out.PendingTools++
	case TranToolCallsDone:
		out.PendingTools--
		if out.PendingTools < 0 {
			out.PendingTools = 0
		}
	case TranStartCompact:
		out.InCompact = true
		out.CompactCount++
		// AutoCompactTracking mirrors autoCompact.ts bookkeeping on compact start (H6).
		out.AutoCompactTracking = &compact.AutoCompactTracking{
			Compacted:   true,
			TurnCounter: out.TurnCount,
			TurnID:      fmt.Sprintf("autocompact:%d", out.CompactCount),
		}
	case TranFinishCompact:
		out.InCompact = false
	default:
		// unknown: no-op
	}
	return out
}

// Continue reasons mirror query.ts State.transition.reason (queryLoop continue sites).
const (
	ContinueReasonCollapseDrainRetry      = "collapse_drain_retry"
	ContinueReasonReactiveCompactRetry    = "reactive_compact_retry"
	ContinueReasonMaxOutputTokensEscalate = "max_output_tokens_escalate"
	ContinueReasonMaxOutputTokensRecovery = "max_output_tokens_recovery"
	ContinueReasonStopHookBlocking        = "stop_hook_blocking"
	// ContinueReasonStopHookPrevented mirrors query.ts terminal stop_hook_prevented (H6 headless).
	ContinueReasonStopHookPrevented       = "stop_hook_prevented"
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

// MirrorAutocompactConsecutiveFailures writes n into st.AutoCompactTracking.ConsecutiveFailures (H3 / autoCompact.ts).
func MirrorAutocompactConsecutiveFailures(st *LoopState, n int) {
	if st == nil {
		return
	}
	if st.AutoCompactTracking == nil {
		st.AutoCompactTracking = &compact.AutoCompactTracking{}
	}
	v := n
	st.AutoCompactTracking.ConsecutiveFailures = &v
}
