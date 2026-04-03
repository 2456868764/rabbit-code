package query

import (
	"encoding/json"
	"strings"
)

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

type autoCompactTrackingJSON struct {
	Compacted           bool   `json:"compacted,omitempty"`
	TurnCounter         int    `json:"turnCounter,omitempty"`
	TurnID              string `json:"turnId,omitempty"`
	ConsecutiveFailures *int   `json:"consecutiveFailures,omitempty"`
}

// MarshalAutoCompactTrackingJSON encodes tracking for session persistence (restore after restart).
func MarshalAutoCompactTrackingJSON(t *AutoCompactTracking) ([]byte, error) {
	if t == nil {
		return []byte("{}"), nil
	}
	j := autoCompactTrackingJSON{
		Compacted:   t.Compacted,
		TurnCounter: t.TurnCounter,
		TurnID:      t.TurnID,
	}
	if t.ConsecutiveFailures != nil {
		v := *t.ConsecutiveFailures
		j.ConsecutiveFailures = &v
	}
	return json.Marshal(j)
}

// UnmarshalAutoCompactTrackingJSON decodes session JSON into AutoCompactTracking (nil input/empty → nil).
func UnmarshalAutoCompactTrackingJSON(data []byte) (*AutoCompactTracking, error) {
	if len(data) == 0 || strings.TrimSpace(string(data)) == "" {
		return nil, nil
	}
	var j autoCompactTrackingJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return nil, err
	}
	out := &AutoCompactTracking{
		Compacted:   j.Compacted,
		TurnCounter: j.TurnCounter,
		TurnID:      j.TurnID,
	}
	if j.ConsecutiveFailures != nil {
		v := *j.ConsecutiveFailures
		out.ConsecutiveFailures = &v
	}
	return out, nil
}
