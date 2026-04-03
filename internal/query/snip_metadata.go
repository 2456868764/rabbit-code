package query

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

// SnipRemovalKind classifies automatic transcript prefix trims (H7 / sessionStorage snip replay).
type SnipRemovalKind string

const (
	SnipRemovalKindHistorySnip SnipRemovalKind = "history_snip"
	SnipRemovalKindSnipCompact SnipRemovalKind = "snip_compact"
)

// SnipRemovalEntry records one trim wave: how many leading messages were dropped and stable id for telemetry / persistence.
// Aligns with sessionStorage.ts boundary snipMetadata.removedUuids semantics in the linear-prefix case.
type SnipRemovalEntry struct {
	ID                  string          `json:"id"`
	Kind                SnipRemovalKind `json:"kind"`
	RemovedMessageCount int             `json:"removedMessageCount"`
	BytesBefore         int             `json:"bytesBefore"`
	BytesAfter          int             `json:"bytesAfter"`
}

// NewSnipRemovalID returns a random 32-char hex id (UUID-sized, no extra deps).
func NewSnipRemovalID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "snip-fallback"
	}
	return hex.EncodeToString(b[:])
}

// CloneSnipRemovalLog returns a deep copy of the log slice (nil if empty).
func CloneSnipRemovalLog(log []SnipRemovalEntry) []SnipRemovalEntry {
	if len(log) == 0 {
		return nil
	}
	return append([]SnipRemovalEntry(nil), log...)
}

// ReplaySnipRemovals applies recorded prefix drops in order to a full transcript (session restore / round-trip).
func ReplaySnipRemovals(msgs json.RawMessage, entries []SnipRemovalEntry) (json.RawMessage, error) {
	out := msgs
	for _, e := range entries {
		if e.RemovedMessageCount <= 0 {
			continue
		}
		next, err := SnipDropFirstMessages(out, e.RemovedMessageCount)
		if err != nil {
			return nil, fmt.Errorf("query: replay snip %s: %w", e.ID, err)
		}
		out = next
	}
	return out, nil
}
