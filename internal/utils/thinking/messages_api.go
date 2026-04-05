package thinking

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
)

// MessagesAPIThinkingField builds the Messages API `thinking` JSON when extended thinking applies,
// mirroring claude.ts ~1596–1629 for inner calls (e.g. WebSearch when not on the plum/Haiku path).
// When hasThinking is false, callers typically send temperature: 1 and omit thinking.
func MessagesAPIThinkingField(model string, p Provider, maxOutputTokens int) (raw json.RawMessage, hasThinking bool, err error) {
	if truthyEnv("RABBIT_CODE_DISABLE_THINKING") {
		return nil, false, nil
	}
	opts := InterleavedAPIContextManagementOpts(model, p)
	if !opts.HasThinking || !ModelSupportsThinking(model, p) {
		return nil, false, nil
	}
	if ModelSupportsAdaptiveThinking(model, p) && !truthyEnv("RABBIT_CODE_DISABLE_ADAPTIVE_THINKING") {
		return json.RawMessage(`{"type":"adaptive"}`), true, nil
	}
	budget := thinkingBudgetTokensFromEnv()
	if maxOutputTokens > 1 && budget > maxOutputTokens-1 {
		budget = maxOutputTokens - 1
	}
	if budget < 1 {
		budget = 1
	}
	raw, err = json.Marshal(map[string]any{
		"type":          "enabled",
		"budget_tokens": budget,
	})
	if err != nil {
		return nil, false, err
	}
	return raw, true, nil
}

func thinkingBudgetTokensFromEnv() int {
	for _, k := range []string{"MAX_THINKING_TOKENS", "RABBIT_CODE_MAX_THINKING_TOKENS"} {
		if v := strings.TrimSpace(os.Getenv(k)); v != "" {
			if n, err := strconv.Atoi(v); err == nil && n > 0 {
				return n
			}
		}
	}
	return 10000
}
