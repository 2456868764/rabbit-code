package querydeps

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
)

// EnsureForkPartialFromForkCompactSummary wires ForkPartialCompactSummary when it is nil and ForkCompactSummary is set.
// The partial stream body is [...contextMessages, summaryUser]; the bridge passes (summaryUser, json(contextMessages))
// to ForkCompactSummary — the same split as full compact's fork path when the host builds one summary user + transcript slice.
// If your fork requires the full session transcript for cache keys, set ForkPartialCompactSummary explicitly instead.
func EnsureForkPartialFromForkCompactSummary(a *AnthropicAssistant) {
	if a == nil || a.ForkPartialCompactSummary != nil || a.ForkCompactSummary == nil {
		return
	}
	full := a.ForkCompactSummary
	a.ForkPartialCompactSummary = func(ctx context.Context, messagesJSON []byte) (string, error) {
		var arr []json.RawMessage
		raw := bytes.TrimSpace(messagesJSON)
		if len(raw) == 0 || string(raw) == "null" {
			return "", errors.New("querydeps: partial fork: empty messages")
		}
		if err := json.Unmarshal(raw, &arr); err != nil {
			return "", err
		}
		if len(arr) < 2 {
			return "", errors.New("querydeps: partial fork: need context messages and summary user")
		}
		sumUser := append(json.RawMessage(nil), arr[len(arr)-1]...)
		prefix, err := json.Marshal(arr[:len(arr)-1])
		if err != nil {
			return "", err
		}
		return full(ctx, sumUser, prefix)
	}
}
