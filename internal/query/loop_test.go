package query

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/query/querydeps"
)

func TestLoopDriver_RunAssistantChain_sequence(t *testing.T) {
	seq := &querydeps.SequenceAssistant{Replies: []string{"first", "second"}}
	d := LoopDriver{
		Deps:      querydeps.Deps{Assistant: seq},
		Model:     "m",
		MaxTokens: 16,
	}
	final, texts, err := d.RunAssistantChain(context.Background(), "hi", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(texts) != 2 || texts[0] != "first" || texts[1] != "second" {
		t.Fatalf("%#v", texts)
	}
	if final == nil {
		t.Fatal("nil final")
	}
}

func TestLoopDriver_RunToolStep_state(t *testing.T) {
	var tools mockToolRunner
	d := LoopDriver{Deps: querydeps.Deps{Tools: tools}}
	st := LoopState{}
	out, err := d.RunToolStep(context.Background(), &st, "bash", []byte(`{}`))
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != `{"ok":true}` {
		t.Fatalf("%s", out)
	}
	if st.PendingTools != 0 || st.TurnCount != 0 {
		t.Fatalf("%+v", st)
	}
}

type mockToolRunner struct{}

func (mockToolRunner) RunTool(context.Context, string, []byte) ([]byte, error) {
	return []byte(`{"ok":true}`), nil
}

func TestLoopDriver_RunTurnLoop_blockingLimitBeforeAssistant(t *testing.T) {
	t.Setenv(features.EnvContextCollapse, "")
	t.Setenv(features.EnvReactiveCompact, "")
	t.Setenv(features.EnvDisableCompact, "")
	t.Setenv(features.EnvDisableAutoCompact, "")
	t.Setenv(features.EnvAutoCompact, "")
	t.Setenv(features.EnvBlockingLimitOverride, "1")
	var calls int
	seq := &querydeps.SequenceTurnAssistant{Turns: []querydeps.TurnResult{{Text: "ok"}}}
	d := LoopDriver{
		Deps: querydeps.Deps{
			Turn: &countingTurnAssistant{inner: seq, after: func() { calls++ }},
		},
		Model:               "m",
		MaxTokens:           8,
		ContextWindowTokens: 50_000,
	}
	st := LoopState{}
	_, _, err := d.RunTurnLoop(context.Background(), &st, strings.Repeat("z", 800))
	if !errors.Is(err, ErrBlockingLimit) {
		t.Fatalf("got %v calls=%d", err, calls)
	}
	if calls != 0 {
		t.Fatalf("expected no AssistantTurn, got %d", calls)
	}
}

type countingTurnAssistant struct {
	inner querydeps.TurnAssistant
	after func()
}

func (c *countingTurnAssistant) AssistantTurn(ctx context.Context, model string, maxTokens int, messagesJSON []byte) (querydeps.TurnResult, error) {
	if c.after != nil {
		c.after()
	}
	return c.inner.AssistantTurn(ctx, model, maxTokens, messagesJSON)
}

func TestLoopDriver_RunTurnLoopFromMessages_skipsBlockingLimit(t *testing.T) {
	t.Setenv(features.EnvContextCollapse, "")
	t.Setenv(features.EnvReactiveCompact, "")
	t.Setenv(features.EnvDisableCompact, "")
	t.Setenv(features.EnvDisableAutoCompact, "")
	t.Setenv(features.EnvAutoCompact, "")
	t.Setenv(features.EnvBlockingLimitOverride, "1")
	var calls int
	seq := &querydeps.SequenceTurnAssistant{Turns: []querydeps.TurnResult{{Text: "ok"}}}
	d := LoopDriver{
		Deps: querydeps.Deps{
			Turn: &countingTurnAssistant{inner: seq, after: func() { calls++ }},
		},
		Model:               "m",
		MaxTokens:           8,
		ContextWindowTokens: 50_000,
	}
	seed, err := InitialUserMessagesJSON(strings.Repeat("z", 800))
	if err != nil {
		t.Fatal(err)
	}
	st := LoopState{}
	_, _, err = d.RunTurnLoopFromMessages(context.Background(), &st, seed)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("expected one AssistantTurn after seed continuation, calls=%d", calls)
	}
}

func TestLoopDriver_RunTurnLoop_preCanceledContext_setsAbortMirror(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	d := LoopDriver{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(ctx context.Context, _ string, _ int, _ []byte) (string, error) {
				return "", ctx.Err()
			}),
		},
		Model: "m", MaxTokens: 8,
	}
	st := LoopState{}
	_, _, err := d.RunTurnLoop(ctx, &st, "hi")
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("got %v", err)
	}
	if !st.ToolUseContext.AbortSignalAborted {
		t.Fatalf("ToolUseContext %+v", st.ToolUseContext)
	}
}
