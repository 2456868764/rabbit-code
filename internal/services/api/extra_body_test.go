package anthropic

import (
	"encoding/json"
	"testing"
)

func TestExtraBodyParamsFromEnv_mergeKeys(t *testing.T) {
	t.Setenv(EnvClaudeCodeExtraBody, `{"top_p":0.9,"foo":1}`)
	t.Setenv(EnvRabbitExtraBody, `{"foo":2,"speed":"fast"}`)
	m := extraBodyParamsFromEnv()
	if m["foo"].(float64) != 2 {
		t.Fatalf("rabbit extra should override: %#v", m["foo"])
	}
	if m["top_p"].(float64) != 0.9 {
		t.Fatal(m)
	}
	if m["speed"] != "fast" {
		t.Fatal(m)
	}
}

func TestMergeAnthropicBetaIntoMap_dedupe(t *testing.T) {
	m := map[string]any{
		"anthropic_beta": []any{"a", "b"},
	}
	mergeExtraBodyIntoMap(m, map[string]any{
		"anthropic_beta": []any{"b", "c"},
	})
	raw, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	var probe struct {
		Beta []string `json:"anthropic_beta"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		t.Fatal(err)
	}
	if len(probe.Beta) != 3 || probe.Beta[0] != "a" || probe.Beta[1] != "b" || probe.Beta[2] != "c" {
		t.Fatalf("%v", probe.Beta)
	}
}

func TestApplyExtraBodyMerge_overridesScalar(t *testing.T) {
	raw := []byte(`{"model":"m","max_tokens":1,"stream":true}`)
	out, err := applyExtraBodyMerge(raw, map[string]any{"max_tokens": 99.0})
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(out) {
		t.Fatal(string(out))
	}
	var m map[string]any
	_ = json.Unmarshal(out, &m)
	if int(m["max_tokens"].(float64)) != 99 {
		t.Fatalf("%v", m["max_tokens"])
	}
}
