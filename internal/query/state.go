package query

// LoopState tracks cross-iteration query loop metadata (subset of query.ts State; item 11 extends parity fields).
type LoopState struct {
	TurnCount    int
	PendingTools int
	InCompact    bool
	// MaxTurns if > 0 caps assistant turns (enforcement in loop later).
	MaxTurns int
	// CompactCount increments on TranStartCompact (P5.1.1 / P5.2.1 bookkeeping).
	CompactCount int
	// MaxOutputTokensRecoveryCount mirrors query.ts maxOutputTokensRecoveryCount (bookkeeping for recovery).
	MaxOutputTokensRecoveryCount int
	// HasAttemptedReactiveCompact mirrors query.ts hasAttemptedReactiveCompact.
	HasAttemptedReactiveCompact bool
	// StopHookActive mirrors query.ts stopHookActive.
	StopHookActive bool
	// Recovery / stream metadata (P5.1.1).
	RecoveryPhase    RecoveryPhase
	RecoveryAttempts int
	LastStopReason   string
	HadStreamError   bool
	// LastAPIErrorKind is the anthropic.APIError kind string after a failed assistant call (P5.1.3).
	LastAPIErrorKind string
}
