package query

// LoopState tracks cross-iteration query loop metadata aligned with query.ts State (H6).
// Messages JSON and full toolUseContext live outside this struct in the Go headless path.
type LoopState struct {
	TurnCount    int
	PendingTools int
	InCompact    bool
	// MaxTurns if > 0 caps assistant turns (enforcement in loop later).
	MaxTurns int
	// CompactCount increments on TranStartCompact (P5.1.1 / P5.2.1 bookkeeping).
	CompactCount int
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
