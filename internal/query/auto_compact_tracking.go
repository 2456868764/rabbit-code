package query

// AutoCompactTracking mirrors services/compact/autoCompact.ts AutoCompactTrackingState (headless bookkeeping, H6).
type AutoCompactTracking struct {
	Compacted           bool
	TurnCounter         int
	TurnID              string
	ConsecutiveFailures *int // nil = field omitted in TS; use pointer for optional
}

// CloneAutoCompactTracking returns a deep copy (ConsecutiveFailures pointer duplicated).
func CloneAutoCompactTracking(p *AutoCompactTracking) *AutoCompactTracking {
	if p == nil {
		return nil
	}
	c := *p
	if p.ConsecutiveFailures != nil {
		v := *p.ConsecutiveFailures
		c.ConsecutiveFailures = &v
	}
	return &c
}
