package anthropic

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
)

func TestEnsureForkPartialFromForkCompactSummary_bridges(t *testing.T) {
	var gotSum, gotTr []byte
	a := &AnthropicAssistant{
		ForkCompactSummary: func(ctx context.Context, summaryUserJSON []byte, transcriptJSON []byte) (string, error) {
			_ = ctx
			gotSum = append([]byte(nil), summaryUserJSON...)
			gotTr = append([]byte(nil), transcriptJSON...)
			return "<s>ok</s>", nil
		},
	}
	EnsureForkPartialFromForkCompactSummary(a)
	if a.ForkPartialCompactSummary == nil {
		t.Fatal("expected ForkPartialCompactSummary")
	}
	msgs, err := json.Marshal([]json.RawMessage{
		json.RawMessage(`{"role":"user","content":[{"type":"text","text":"ctx"}]}`),
		json.RawMessage(`{"role":"user","content":[{"type":"text","text":"sum"}]}`),
	})
	if err != nil {
		t.Fatal(err)
	}
	out, err := a.ForkPartialCompactSummary(context.Background(), msgs)
	if err != nil {
		t.Fatal(err)
	}
	if out != "<s>ok</s>" {
		t.Fatalf("out %q", out)
	}
	if !bytes.Contains(gotSum, []byte("sum")) {
		t.Fatalf("summary user %s", gotSum)
	}
	if !bytes.Contains(gotTr, []byte("ctx")) {
		t.Fatalf("transcript %s", gotTr)
	}
}

func TestEnsureForkPartialFromForkCompactSummary_idempotentWhenPartialSet(t *testing.T) {
	calls := 0
	a := &AnthropicAssistant{
		ForkCompactSummary: func(context.Context, []byte, []byte) (string, error) {
			calls++
			return "", nil
		},
		ForkPartialCompactSummary: func(context.Context, []byte) (string, error) { return "p", nil },
	}
	EnsureForkPartialFromForkCompactSummary(a)
	if calls != 0 {
		t.Fatal("ForkCompactSummary should not run")
	}
	s, err := a.ForkPartialCompactSummary(context.Background(), []byte(`[{},{}]`))
	if err != nil || s != "p" {
		t.Fatalf("got %q %v", s, err)
	}
}
