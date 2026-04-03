package query

import "encoding/json"

// ToolUseContextMirror holds a headless subset of query.ts ToolUseContext (H6).
// Full TS type includes AppState, MCP, hooks, etc.; Go mirrors fields used for token-budget / analytics parity.
type ToolUseContextMirror struct {
	AgentID        string
	MainLoopModel  string
	NonInteractive bool
	QueryChainID   string
	QueryDepth     int
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
	AutoCompactTracking *AutoCompactTracking
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
