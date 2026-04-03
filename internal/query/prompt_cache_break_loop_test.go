package query

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"github.com/2456868764/rabbit-code/internal/anthropic"
	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/querydeps"
)

type stripVerifyTurnAssistant struct {
	n int
	t *testing.T
}

func (s *stripVerifyTurnAssistant) AssistantTurn(ctx context.Context, model string, maxTokens int, msgs []byte) (querydeps.TurnResult, error) {
	s.n++
	if s.n == 1 {
		return querydeps.TurnResult{}, anthropic.ErrPromptCacheBreakDetected
	}
	if len(msgs) == 0 || bytes.Contains(msgs, []byte("cache_control")) {
		s.t.Fatalf("call %d: expected stripped messages, got %s", s.n, msgs)
	}
	return querydeps.TurnResult{Text: "ok"}, nil
}

func TestRunTurnLoop_promptCacheBreak_trimResend(t *testing.T) {
	t.Setenv(features.EnvPromptCacheBreak, "1")
	t.Setenv(features.EnvPromptCacheBreakTrimResend, "1")
	seed := json.RawMessage(`[{"role":"user","content":[{"type":"text","text":"hi","cache_control":{"type":"ephemeral"}}]}]`)
	d := LoopDriver{
		Deps: querydeps.Deps{
			Turn: &stripVerifyTurnAssistant{t: t},
		},
		Model: "m", MaxTokens: 8,
	}
	st := LoopState{}
	_, _, err := d.RunTurnLoopFromMessages(context.Background(), &st, seed)
	if err != nil {
		t.Fatal(err)
	}
	if st.LoopContinue.Reason != ContinueReasonPromptCacheBreakTrimResend {
		t.Fatalf("continue: %+v", st.LoopContinue)
	}
}

type failOnceCacheBreakTurn struct {
	n int
}

func (f *failOnceCacheBreakTurn) AssistantTurn(ctx context.Context, model string, maxTokens int, msgs []byte) (querydeps.TurnResult, error) {
	f.n++
	if f.n == 1 {
		return querydeps.TurnResult{}, anthropic.ErrPromptCacheBreakDetected
	}
	return querydeps.TurnResult{Text: "recovered"}, nil
}

func TestRunTurnLoop_promptCacheBreak_compactRecovery(t *testing.T) {
	t.Setenv(features.EnvPromptCacheBreak, "1")
	t.Setenv(features.EnvPromptCacheBreakTrimResend, "0")
	t.Setenv(features.EnvPromptCacheBreakAutoCompact, "1")
	nextTranscript := []byte(`[{"role":"user","content":[{"type":"text","text":"compact-seed"}]}]`)
	d := LoopDriver{
		Deps: querydeps.Deps{
			Turn: &failOnceCacheBreakTurn{},
		},
		Model: "m", MaxTokens: 8,
		PromptCacheBreakRecovery: func(ctx context.Context, msgs json.RawMessage) (json.RawMessage, bool, error) {
			_ = msgs
			return json.RawMessage(append([]byte(nil), nextTranscript...)), true, nil
		},
	}
	st := LoopState{}
	_, text, err := d.RunTurnLoop(context.Background(), &st, "hi")
	if err != nil {
		t.Fatal(err)
	}
	if text != "recovered" {
		t.Fatalf("text %q", text)
	}
	if st.LoopContinue.Reason != ContinueReasonPromptCacheBreakCompactRetry {
		t.Fatalf("continue: %+v", st.LoopContinue)
	}
}

type stripErrorTurnAssistant struct{}

func (stripErrorTurnAssistant) AssistantTurn(ctx context.Context, model string, maxTokens int, msgs []byte) (querydeps.TurnResult, error) {
	return querydeps.TurnResult{}, anthropic.ErrPromptCacheBreakDetected
}

func TestRunTurnLoop_promptCacheBreak_stripJSONError(t *testing.T) {
	t.Setenv(features.EnvPromptCacheBreak, "1")
	t.Setenv(features.EnvPromptCacheBreakTrimResend, "1")
	// Invalid messages JSON: strip fails before retry; must not swallow error as generic cache break.
	seed := json.RawMessage(`not-a-json-array`)
	d := LoopDriver{
		Deps: querydeps.Deps{
			Turn: stripErrorTurnAssistant{},
		},
		Model: "m", MaxTokens: 8,
	}
	st := LoopState{}
	_, _, err := d.RunTurnLoopFromMessages(context.Background(), &st, seed)
	if err == nil {
		t.Fatal("expected error")
	}
}

type trimThenCompactTurnAssistant struct {
	n int
	t *testing.T
}

func (a *trimThenCompactTurnAssistant) AssistantTurn(ctx context.Context, model string, maxTokens int, msgs []byte) (querydeps.TurnResult, error) {
	a.n++
	switch a.n {
	case 1:
		return querydeps.TurnResult{}, anthropic.ErrPromptCacheBreakDetected
	case 2:
		if bytes.Contains(msgs, []byte("cache_control")) {
			a.t.Fatalf("call 2: expected stripped msgs")
		}
		return querydeps.TurnResult{}, anthropic.ErrPromptCacheBreakDetected
	default:
		return querydeps.TurnResult{Text: "after-compact"}, nil
	}
}

func TestRunTurnLoop_promptCacheBreak_trimThenCompact_chain(t *testing.T) {
	t.Setenv(features.EnvPromptCacheBreak, "1")
	t.Setenv(features.EnvPromptCacheBreakTrimResend, "1")
	t.Setenv(features.EnvPromptCacheBreakAutoCompact, "1")
	seed := json.RawMessage(`[{"role":"user","content":[{"type":"text","text":"x","cache_control":{"type":"ephemeral"}}]}]`)
	turn := &trimThenCompactTurnAssistant{t: t}
	d := LoopDriver{
		Deps: querydeps.Deps{
			Turn: turn,
		},
		Model: "m", MaxTokens: 8,
		PromptCacheBreakRecovery: func(ctx context.Context, msgs json.RawMessage) (json.RawMessage, bool, error) {
			_ = ctx
			_ = msgs
			return json.RawMessage(`[{"role":"user","content":[{"type":"text","text":"c"}]}]`), true, nil
		},
	}
	st := LoopState{}
	_, _, err := d.RunTurnLoopFromMessages(context.Background(), &st, seed)
	if err != nil {
		t.Fatal(err)
	}
	if st.LoopContinue.Reason != ContinueReasonPromptCacheBreakCompactRetry {
		t.Fatalf("want last continue compact_retry, got %+v", st.LoopContinue)
	}
	if turn.n != 3 {
		t.Fatalf("AssistantTurn calls: want 3 (cache break → strip retry cache break → compact seed ok), got %d", turn.n)
	}
}
