package query

import "encoding/json"

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
