package query

import (
	"encoding/json"
	"fmt"
)

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
