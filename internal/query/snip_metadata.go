package query

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
)

// SnipRemovalKind classifies automatic transcript prefix trims (H7 / sessionStorage snip replay).
type SnipRemovalKind string

const (
	SnipRemovalKindHistorySnip SnipRemovalKind = "history_snip"
	SnipRemovalKindSnipCompact SnipRemovalKind = "snip_compact"
)

// SnipRemovalEntry records one trim wave: prefix drop and/or index/uuid-based removals for session replay (H7).
// Priority when replaying: RemovedIndices > RemovedUUIDs (with map) > RemovedMessageCount (prefix only).
// Aligns with sessionStorage.ts snipMetadata.removedUuids when UUIDToIndex is provided to ReplaySnipRemovalsEx.
type SnipRemovalEntry struct {
	ID                  string          `json:"id"`
	Kind                SnipRemovalKind `json:"kind"`
	RemovedMessageCount int             `json:"removedMessageCount"`
	BytesBefore         int             `json:"bytesBefore"`
	BytesAfter          int             `json:"bytesAfter"`
	// RemovedIndices: 0-based message indices to delete (middle snip / arbitrary positions). Applied in descending order.
	RemovedIndices []int `json:"removedIndices,omitempty"`
	// RemovedUUIDs: TS sessionStorage interchange; requires SnipReplayOptions.UUIDToIndex in ReplaySnipRemovalsEx.
	RemovedUUIDs []string `json:"removedUuids,omitempty"`
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

// SnipReplayOptions configures ReplaySnipRemovalsEx (optional UUID→index map for TS removedUuids replay).
type SnipReplayOptions struct {
	UUIDToIndex map[string]int
}

// ReplaySnipRemovals applies recorded removals in order (prefix-only entries; entries with removedUuids alone error).
func ReplaySnipRemovals(msgs json.RawMessage, entries []SnipRemovalEntry) (json.RawMessage, error) {
	return ReplaySnipRemovalsEx(msgs, entries, nil)
}

// ReplaySnipRemovalsEx replays snip log: per entry, removes by indices, or resolves UUIDs with opt.UUIDToIndex, or prefix drop.
func ReplaySnipRemovalsEx(msgs json.RawMessage, entries []SnipRemovalEntry, opt *SnipReplayOptions) (json.RawMessage, error) {
	out := msgs
	for _, e := range entries {
		var next json.RawMessage
		var err error
		switch {
		case len(e.RemovedIndices) > 0:
			next, err = snipRemoveMessageIndices(out, e.RemovedIndices)
		case len(e.RemovedUUIDs) > 0:
			if opt == nil || len(opt.UUIDToIndex) == 0 {
				return nil, fmt.Errorf("query: replay snip %s: %w", e.ID, ErrSnipReplayUUIDMapRequired)
			}
			idx := uuidListToIndices(e.RemovedUUIDs, opt.UUIDToIndex)
			if len(idx) == 0 {
				continue
			}
			next, err = snipRemoveMessageIndices(out, idx)
		case e.RemovedMessageCount > 0:
			next, err = SnipDropFirstMessages(out, e.RemovedMessageCount)
		default:
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("query: replay snip %s: %w", e.ID, err)
		}
		out = next
	}
	return out, nil
}

func uuidListToIndices(uuids []string, m map[string]int) []int {
	seenIdx := make(map[int]struct{})
	for _, u := range uuids {
		if i, ok := m[u]; ok {
			seenIdx[i] = struct{}{}
		}
	}
	out := make([]int, 0, len(seenIdx))
	for i := range seenIdx {
		out = append(out, i)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(out)))
	return out
}

func snipRemoveMessageIndices(msgs json.RawMessage, indices []int) (json.RawMessage, error) {
	var arr []json.RawMessage
	if err := json.Unmarshal(msgs, &arr); err != nil {
		return nil, err
	}
	uniq := make(map[int]struct{})
	for _, i := range indices {
		uniq[i] = struct{}{}
	}
	var sorted []int
	for i := range uniq {
		sorted = append(sorted, i)
	}
	sort.Sort(sort.Reverse(sort.IntSlice(sorted)))
	for _, i := range sorted {
		if i < 0 || i >= len(arr) {
			return nil, fmt.Errorf("query: snip index %d out of range (len=%d)", i, len(arr))
		}
		arr = append(arr[:i], arr[i+1:]...)
	}
	return json.Marshal(arr)
}

// MergeSnipRemovalLogs appends b to a, skipping entries whose ID already appears in a (H7 session merge).
func MergeSnipRemovalLogs(a, b []SnipRemovalEntry) []SnipRemovalEntry {
	seen := make(map[string]struct{})
	out := make([]SnipRemovalEntry, 0, len(a)+len(b))
	for _, e := range a {
		out = append(out, e)
		if e.ID != "" {
			seen[e.ID] = struct{}{}
		}
	}
	for _, e := range b {
		if e.ID != "" {
			if _, ok := seen[e.ID]; ok {
				continue
			}
			seen[e.ID] = struct{}{}
		}
		out = append(out, e)
	}
	return out
}
