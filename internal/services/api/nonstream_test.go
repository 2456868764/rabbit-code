package anthropic

import (
	"encoding/json"
	"testing"
)

func TestAdjustMessagesStreamBodyForNonStreaming_capsMaxAndThinking(t *testing.T) {
	th := []byte(`{"type":"enabled","budget_tokens":100000}`)
	body := MessagesStreamBody{Model: "m", MaxTokens: 200_000, Thinking: th}
	out := AdjustMessagesStreamBodyForNonStreaming(body)
	if out.MaxTokens != MaxNonStreamingTokens {
		t.Fatalf("max_tokens=%d", out.MaxTokens)
	}
	var probe struct {
		Budget int `json:"budget_tokens"`
	}
	if err := json.Unmarshal(out.Thinking, &probe); err != nil {
		t.Fatal(err)
	}
	if probe.Budget != out.MaxTokens-1 {
		t.Fatalf("budget_tokens=%d want %d", probe.Budget, out.MaxTokens-1)
	}
}

func TestDecodeNonStreamingMessageResponse_textAndUsage(t *testing.T) {
	raw := []byte(`{"content":[{"type":"text","text":"hi"}],"usage":{"input_tokens":1,"output_tokens":2}}`)
	text, u, err := DecodeNonStreamingMessageResponse(raw)
	if err != nil || text != "hi" || u.InputTokens != 1 || u.OutputTokens != 2 {
		t.Fatalf("got %q %+v err=%v", text, u, err)
	}
}
