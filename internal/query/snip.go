package query

import (
	"encoding/json"
	"errors"
)

// ErrSnipInvalidN is returned when n is negative for snip helpers.
var ErrSnipInvalidN = errors.New("query: snip n must be non-negative")

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
