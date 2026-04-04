package query

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
)

// ErrSnipInvalidN is returned when n is negative for snip helpers.
var ErrSnipInvalidN = errors.New("query: snip n must be non-negative")

// ErrSnipReplayUUIDMapRequired is returned when a log entry carries removedUuids but ReplaySnipRemovals has no map to resolve them.
var ErrSnipReplayUUIDMapRequired = errors.New("query: snip entry uses removedUuids; use ReplaySnipRemovalsEx with SnipReplayOptions.UUIDToIndex")

// ErrSnipNoEmbeddedUUIDs means ReplaySnipRemovalsAuto could not find rabbit_message_uuid (or custom field) on messages.
var ErrSnipNoEmbeddedUUIDs = errors.New("query: no embedded message UUID fields for removedUuids replay")

// SnipDropFirstMessages removes the first n elements from a top-level JSON array of Messages-API-style
// message objects (P5.2.2 transcript trim; parity with services/compact/snip-style prefix removal).
// If n >= len(messages), the result is an empty array [].
func SnipDropFirstMessages(messagesJSON json.RawMessage, n int) (json.RawMessage, error) {
	if n < 0 {
		return nil, ErrSnipInvalidN
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(messagesJSON, &arr); err != nil {
		return nil, err
	}
	if n > len(arr) {
		n = len(arr)
	}
	arr = arr[n:]
	return json.Marshal(arr)
}

// MarshalSnipRemovalLogJSON encodes the snip log for session sidecar persistence (H7).
func MarshalSnipRemovalLogJSON(log []SnipRemovalEntry) ([]byte, error) {
	if len(log) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(log)
}

// UnmarshalSnipRemovalLogJSON decodes session JSON; empty or "null" yields nil.
func UnmarshalSnipRemovalLogJSON(data []byte) ([]SnipRemovalEntry, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var log []SnipRemovalEntry
	if err := json.Unmarshal(data, &log); err != nil {
		return nil, err
	}
	return log, nil
}

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

// RabbitMessageUUIDKey is the default JSON object key for per-message stable ids (H7 sidecar; strip before API if needed).
const RabbitMessageUUIDKey = "rabbit_message_uuid"

// BuildUUIDToIndexFromMessagesJSON maps each non-empty string at fieldName on top-level message objects to its array index.
// Duplicate values return an error. Empty fieldName defaults to RabbitMessageUUIDKey.
func BuildUUIDToIndexFromMessagesJSON(msgs json.RawMessage, fieldName string) (map[string]int, error) {
	if len(fieldName) == 0 {
		fieldName = RabbitMessageUUIDKey
	}
	var arr []map[string]json.RawMessage
	if err := json.Unmarshal(msgs, &arr); err != nil {
		return nil, err
	}
	out := make(map[string]int)
	for i, m := range arr {
		raw, ok := m[fieldName]
		if !ok || len(raw) == 0 || raw[0] != '"' {
			continue
		}
		var s string
		if err := json.Unmarshal(raw, &s); err != nil || s == "" {
			continue
		}
		if prev, dup := out[s]; dup {
			return nil, fmt.Errorf("query: duplicate %s %q at indices %d and %d", fieldName, s, prev, i)
		}
		out[s] = i
	}
	return out, nil
}

// StripMessageFieldFromTranscriptJSON removes fieldName from each top-level message object (copies array; preserves other keys).
func StripMessageFieldFromTranscriptJSON(msgs json.RawMessage, fieldName string) (json.RawMessage, error) {
	if fieldName == "" {
		return msgs, nil
	}
	var arr []map[string]json.RawMessage
	if err := json.Unmarshal(msgs, &arr); err != nil {
		return nil, err
	}
	for i := range arr {
		delete(arr[i], fieldName)
	}
	return json.Marshal(arr)
}

// ReplaySnipRemovalsAuto runs ReplaySnipRemovalsEx, building UUIDToIndex from embedded message fields when any entry uses removedUuids.
func ReplaySnipRemovalsAuto(msgs json.RawMessage, entries []SnipRemovalEntry, uuidFieldName string) (json.RawMessage, error) {
	needsMap := false
	for _, e := range entries {
		if len(e.RemovedUUIDs) > 0 {
			needsMap = true
			break
		}
	}
	var opt *SnipReplayOptions
	if needsMap {
		m, err := BuildUUIDToIndexFromMessagesJSON(msgs, uuidFieldName)
		if err != nil {
			return nil, err
		}
		if len(m) == 0 {
			return nil, ErrSnipNoEmbeddedUUIDs
		}
		opt = &SnipReplayOptions{UUIDToIndex: m}
	}
	return ReplaySnipRemovalsEx(msgs, entries, opt)
}

// TranscriptMessageCount returns the number of top-level objects in a Messages API JSON array.
func TranscriptMessageCount(msgs json.RawMessage) (int, error) {
	var arr []json.RawMessage
	if err := json.Unmarshal(msgs, &arr); err != nil {
		return 0, err
	}
	return len(arr), nil
}

// AnnotateTranscriptWithUUIDs sets fieldName (default RabbitMessageUUIDKey) on each top-level message to uuids[i].
// len(uuids) must equal the message count.
func AnnotateTranscriptWithUUIDs(msgs json.RawMessage, uuids []string, fieldName string) (json.RawMessage, error) {
	if len(fieldName) == 0 {
		fieldName = RabbitMessageUUIDKey
	}
	var arr []map[string]json.RawMessage
	if err := json.Unmarshal(msgs, &arr); err != nil {
		return nil, err
	}
	if len(uuids) != len(arr) {
		return nil, fmt.Errorf("query: annotate uuid: got %d uuids for %d messages", len(uuids), len(arr))
	}
	for i := range arr {
		raw, err := json.Marshal(uuids[i])
		if err != nil {
			return nil, err
		}
		arr[i][fieldName] = raw
	}
	return json.Marshal(arr)
}

// StripMessageFieldsFromTranscriptJSON removes each non-empty field name from every top-level message (order preserved).
func StripMessageFieldsFromTranscriptJSON(msgs json.RawMessage, fieldNames []string) (json.RawMessage, error) {
	out := msgs
	var err error
	for _, f := range fieldNames {
		if f == "" {
			continue
		}
		out, err = StripMessageFieldFromTranscriptJSON(out, f)
		if err != nil {
			return nil, err
		}
	}
	return out, nil
}
