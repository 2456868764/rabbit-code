package query

import (
	"encoding/json"
)

// TrimTranscriptPrefixWhileOverBudget drops leading messages (one per round) while len(msgs) > maxBytes.
// maxRounds caps how many SnipDropFirstMessages calls run (avoid infinite loop). Returns rounds applied.
func TrimTranscriptPrefixWhileOverBudget(msgs json.RawMessage, maxBytes, maxRounds int) (out json.RawMessage, rounds int, err error) {
	out = msgs
	if maxBytes <= 0 || maxRounds <= 0 {
		return out, 0, nil
	}
	for len(out) > maxBytes && maxRounds > 0 {
		next, err := SnipDropFirstMessages(out, 1)
		if err != nil {
			return msgs, rounds, err
		}
		var arr []json.RawMessage
		if err := json.Unmarshal(next, &arr); err != nil {
			return msgs, rounds, err
		}
		if len(arr) == 0 {
			break
		}
		out = next
		rounds++
		maxRounds--
	}
	return out, rounds, nil
}
