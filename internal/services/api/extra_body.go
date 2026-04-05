package anthropic

import (
	"encoding/json"
	"os"
	"strings"
)

// EnvClaudeCodeExtraBody mirrors process.env.CLAUDE_CODE_EXTRA_BODY (claude.ts getExtraBodyParams).
const EnvClaudeCodeExtraBody = "CLAUDE_CODE_EXTRA_BODY"

// EnvRabbitExtraBody is merged after EnvClaudeCodeExtraBody (rabbit-local override).
const EnvRabbitExtraBody = "RABBIT_CODE_EXTRA_BODY"

// extraBodyParamsFromEnv parses CLAUDE_CODE_EXTRA_BODY then RABBIT_CODE_EXTRA_BODY shallow-merge (latter wins on duplicate keys).
func extraBodyParamsFromEnv() map[string]any {
	out := make(map[string]any)
	for _, envKey := range []string{EnvClaudeCodeExtraBody, EnvRabbitExtraBody} {
		s := strings.TrimSpace(os.Getenv(envKey))
		if s == "" {
			continue
		}
		var parsed map[string]any
		if err := json.Unmarshal([]byte(s), &parsed); err != nil || parsed == nil {
			continue
		}
		for k, v := range parsed {
			out[k] = v
		}
	}
	return out
}

// mergeExtraBodyIntoMap applies getExtraBodyParams parity: keys from env overlay the marshaled request;
// anthropic_beta arrays are merged deduped like claude.ts.
func mergeExtraBodyIntoMap(m map[string]any, extra map[string]any) {
	if len(extra) == 0 || m == nil {
		return
	}
	for k, v := range extra {
		if k == "anthropic_beta" {
			mergeAnthropicBetaIntoMap(m, v)
			continue
		}
		m[k] = v
	}
}

func mergeAnthropicBetaIntoMap(m map[string]any, v any) {
	newNames := normalizeStringSliceFromJSON(v)
	if len(newNames) == 0 {
		return
	}
	existing := normalizeStringSliceFromJSON(m["anthropic_beta"])
	seen := make(map[string]struct{}, len(existing)+len(newNames))
	var merged []string
	for _, s := range existing {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		merged = append(merged, s)
	}
	for _, s := range newNames {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		merged = append(merged, s)
	}
	if len(merged) == 0 {
		return
	}
	out := make([]any, len(merged))
	for i, s := range merged {
		out[i] = s
	}
	m["anthropic_beta"] = out
}

func normalizeStringSliceFromJSON(v any) []string {
	switch x := v.(type) {
	case nil:
		return nil
	case []string:
		return append([]string(nil), x...)
	case []any:
		var out []string
		for _, e := range x {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}
