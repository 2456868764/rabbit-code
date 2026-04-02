package query

// LoopState is a minimal subset of query.ts State (more fields in later commits).
type LoopState struct {
	TurnCount    int
	PendingTools int
}
