package query

// LoopState is a minimal subset of query.ts State (more fields in later commits).
type LoopState struct {
	TurnCount    int
	PendingTools int
	InCompact    bool
	// MaxTurns if > 0 caps assistant turns (enforcement in loop later).
	MaxTurns int
	// CompactCount increments on TranStartCompact (P5.1.1 / P5.2.1 bookkeeping).
	CompactCount int
	// Recovery / stream metadata (P5.1.1).
	RecoveryPhase    RecoveryPhase
	RecoveryAttempts int
	LastStopReason   string
	HadStreamError   bool
	// LastAPIErrorKind is the anthropic.APIError kind string after a failed assistant call (P5.1.3).
	LastAPIErrorKind string
}
