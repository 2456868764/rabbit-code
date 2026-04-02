package phase5cli

import (
	"encoding/json"
	"io"
)

// BreakCacheCommandPayload matches the headless EventKindBreakCacheCommand shape (PhaseDetail "submit").
type BreakCacheCommandPayload struct {
	Kind  string `json:"kind"`
	Phase string `json:"phase"`
}

// WriteBreakCacheCommandJSON prints one JSON line for scripting parity with engine break-cache events (P5.F.6).
func WriteBreakCacheCommandJSON(w io.Writer) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(BreakCacheCommandPayload{Kind: "break_cache_command", Phase: "submit"})
}
