package query

// AutoCompactTracking mirrors services/compact/autoCompact.ts AutoCompactTrackingState (headless bookkeeping, H6).
type AutoCompactTracking struct {
	Compacted           bool
	TurnCounter         int
	TurnID              string
	ConsecutiveFailures *int // nil = field omitted in TS; use pointer for optional
}
