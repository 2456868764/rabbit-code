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

// MirrorAutocompactConsecutiveFailures writes n into st.AutoCompactTracking.ConsecutiveFailures (H3 / autoCompact.ts
// tracking.consecutiveFailures). Used so LoopState reflects the same count as the Engine holds across executor outcomes.
func MirrorAutocompactConsecutiveFailures(st *LoopState, n int) {
	if st == nil {
		return
	}
	if st.AutoCompactTracking == nil {
		st.AutoCompactTracking = &AutoCompactTracking{}
	}
	v := n
	st.AutoCompactTracking.ConsecutiveFailures = &v
}
