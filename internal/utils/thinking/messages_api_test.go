package thinking

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestMessagesAPIThinkingField_sonnet4Budget(t *testing.T) {
	raw, has, err := MessagesAPIThinkingField("claude-sonnet-4-20250514", ProviderAnthropic, 8192)
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal("want thinking for sonnet-4")
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	if m["type"] != "enabled" {
		t.Fatalf("type %v", m["type"])
	}
	if _, ok := m["budget_tokens"]; !ok {
		t.Fatal("missing budget_tokens")
	}
}

func TestMessagesAPIThinkingField_haikuOff(t *testing.T) {
	_, has, err := MessagesAPIThinkingField("claude-3-5-haiku-20241022", ProviderAnthropic, 4096)
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Fatal("haiku-3.5 should not use extended thinking field")
	}
}

func TestMessagesAPIThinkingField_disableThinkingEnv(t *testing.T) {
	t.Setenv("RABBIT_CODE_DISABLE_THINKING", "1")
	defer t.Setenv("RABBIT_CODE_DISABLE_THINKING", "")
	_, has, err := MessagesAPIThinkingField("claude-sonnet-4-20250514", ProviderAnthropic, 8192)
	if err != nil {
		t.Fatal(err)
	}
	if has {
		t.Fatal("env should disable")
	}
}

func TestMessagesAPIThinkingField_adaptive4_6(t *testing.T) {
	raw, has, err := MessagesAPIThinkingField("claude-sonnet-4-6-20250901", ProviderAnthropic, 8192)
	if err != nil {
		t.Fatal(err)
	}
	if !has {
		t.Fatal("want adaptive for 4.6")
	}
	if !strings.Contains(string(raw), "adaptive") {
		t.Fatalf("got %s", raw)
	}
}
