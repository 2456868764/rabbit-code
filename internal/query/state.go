package query

// LoopState is a minimal subset of query.ts State (more fields in later commits).
type LoopState struct {
	TurnCount    int
	PendingTools int
	InCompact    bool
	// MaxTurns if > 0 caps assistant turns (enforcement in loop later).
	MaxTurns int
}
