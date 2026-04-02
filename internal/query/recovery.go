package query

// RecoveryPhase is a coarse recovery lane for LoopState (P5.1.1; full query.ts parity in later phases).
type RecoveryPhase uint8

const (
	RecoveryNone RecoveryPhase = iota
	RecoveryPendingCompact
	RecoveryRetriedOnce
)
