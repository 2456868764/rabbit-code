package query

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/services/api"
)

func TestLoopDriver_RunAssistantChain_sequence(t *testing.T) {
	seq := &SequenceAssistant{Replies: []string{"first", "second"}}
	d := LoopDriver{
		Deps:      Deps{Assistant: seq},
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
	d := LoopDriver{Deps: Deps{Tools: tools}}
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

type errToolRunner struct{}

func (errToolRunner) RunTool(context.Context, string, []byte) ([]byte, error) {
	return nil, errors.New("tool boom")
}

func TestLoopDriver_RunToolStep_toolErrorUndoesSchedule(t *testing.T) {
	d := LoopDriver{Deps: Deps{Tools: errToolRunner{}}}
	st := LoopState{}
	_, err := d.RunToolStep(context.Background(), &st, "bash", []byte(`{}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if st.PendingTools != 0 {
		t.Fatalf("pending=%d", st.PendingTools)
	}
}

func TestLoopContinue_recordAndClear(t *testing.T) {
	var st LoopState
	RecordLoopContinue(&st, LoopContinue{Reason: ContinueReasonReactiveCompactRetry})
	if st.LoopContinue.Reason != ContinueReasonReactiveCompactRetry {
		t.Fatal()
	}
	ClearLoopContinue(&st)
	if !st.LoopContinue.Empty() {
		t.Fatal()
	}
}

func TestLoopDriver_RunTurnLoop_setsNextTurnContinueAfterTools(t *testing.T) {
	turns := &SequenceTurnAssistant{Turns: []TurnResult{
		{
			Text:     "t",
			ToolUses: []ToolUseCall{{ID: "1", Name: "bash", Input: json.RawMessage(`{}`)}},
		},
		{Text: "done"},
	}}
	d := LoopDriver{
		Deps: Deps{
			Tools: BashStubToolRunner{},
			Turn:  turns,
		},
		Model: "m", MaxTokens: 8,
	}
	st := LoopState{}
	_, _, err := d.RunTurnLoop(context.Background(), &st, "hi")
	if err != nil {
		t.Fatal(err)
	}
	if st.LoopContinue.Reason != ContinueReasonNextTurn {
		t.Fatalf("want next_turn, got %+v", st.LoopContinue)
	}
	if len(st.MessagesJSON) == 0 || !strings.Contains(string(st.MessagesJSON), "hi") {
		t.Fatalf("MessagesJSON mirror: %s", st.MessagesJSON)
	}
	if st.ToolUseContext.MainLoopModel != "m" {
		t.Fatalf("ToolUseContext: %+v", st.ToolUseContext)
	}
}

func TestResetLoopStateFieldsForNextQueryIteration_clearsRecoveryAndOverride(t *testing.T) {
	st := LoopState{
		MaxOutputTokensRecoveryCount:  9,
		HasAttemptedReactiveCompact:   true,
		MaxOutputTokensOverrideActive: true,
		MaxOutputTokensOverride:       4096,
	}
	ResetLoopStateFieldsForNextQueryIteration(&st)
	if st.MaxOutputTokensRecoveryCount != 0 || st.HasAttemptedReactiveCompact {
		t.Fatalf("got %+v", st)
	}
	if st.MaxOutputTokensOverrideActive || st.MaxOutputTokensOverride != 0 {
		t.Fatalf("override: active=%v val=%d", st.MaxOutputTokensOverrideActive, st.MaxOutputTokensOverride)
	}
}

func TestLoopDriver_RunTurnLoop_resetsRecoveryFieldsAfterToolRound(t *testing.T) {
	turns := &SequenceTurnAssistant{Turns: []TurnResult{
		{
			Text:     "t",
			ToolUses: []ToolUseCall{{ID: "1", Name: "bash", Input: json.RawMessage(`{}`)}},
		},
		{Text: "done"},
	}}
	d := LoopDriver{
		Deps: Deps{
			Tools: BashStubToolRunner{},
			Turn:  turns,
		},
		Model: "m", MaxTokens: 8,
	}
	st := LoopState{
		MaxOutputTokensRecoveryCount:  7,
		HasAttemptedReactiveCompact:   true,
		MaxOutputTokensOverrideActive: true,
		MaxOutputTokensOverride:       2048,
	}
	_, _, err := d.RunTurnLoop(context.Background(), &st, "hi")
	if err != nil {
		t.Fatal(err)
	}
	if st.MaxOutputTokensRecoveryCount != 0 {
		t.Fatalf("want recovery count 0 after next_turn reset, got %d", st.MaxOutputTokensRecoveryCount)
	}
	if st.HasAttemptedReactiveCompact {
		t.Fatal("want HasAttemptedReactiveCompact false")
	}
	if st.MaxOutputTokensOverrideActive || st.MaxOutputTokensOverride != 0 {
		t.Fatalf("want override cleared, got active=%v val=%d", st.MaxOutputTokensOverrideActive, st.MaxOutputTokensOverride)
	}
}

func TestLoopDriver_RunTurnLoop_noTools_doesNotSetNextTurn(t *testing.T) {
	turns := &SequenceTurnAssistant{Turns: []TurnResult{{Text: "only"}}}
	d := LoopDriver{Deps: Deps{Turn: turns}, Model: "m", MaxTokens: 8}
	st := LoopState{}
	_, _, err := d.RunTurnLoop(context.Background(), &st, "hi")
	if err != nil {
		t.Fatal(err)
	}
	if !st.LoopContinue.Empty() {
		t.Fatalf("expected empty continue, got %+v", st.LoopContinue)
	}
}

type countingToolRunner struct {
	n  int
	mu sync.Mutex
}

func (c *countingToolRunner) RunTool(context.Context, string, []byte) ([]byte, error) {
	c.mu.Lock()
	c.n++
	c.mu.Unlock()
	return []byte(`{"ok":true}`), nil
}

func (c *countingToolRunner) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.n
}

func TestLoopDriver_RunTurnLoop_toolThenText_AC5_3(t *testing.T) {
	tr := &countingToolRunner{}
	turns := &SequenceTurnAssistant{Turns: []TurnResult{
		{
			Text: "invoke",
			ToolUses: []ToolUseCall{
				{ID: "id1", Name: "bash", Input: json.RawMessage(`{"cmd":"ls"}`)},
			},
		},
		{Text: "done"},
	}}
	d := LoopDriver{
		Deps: Deps{
			Tools: tr,
			Turn:  turns,
		},
		Model:          "m",
		MaxTokens:      64,
		AgentID:        "agent-test",
		NonInteractive: true,
		SessionID:      "sess-1",
		Debug:          true,
		QuerySource:    "compact_agent",
	}
	st := LoopState{MaxTurns: 10}
	_, last, err := d.RunTurnLoop(context.Background(), &st, "hi")
	if err != nil {
		t.Fatal(err)
	}
	if last != "done" {
		t.Fatalf("last %q", last)
	}
	if n := tr.count(); n != 1 {
		t.Fatalf("tool runs %d", n)
	}
	if st.TurnCount != 2 {
		t.Fatalf("turns %+v", st)
	}
	if st.ToolUseContext.AgentID != "agent-test" || !st.ToolUseContext.NonInteractive || st.ToolUseContext.MainLoopModel != "m" {
		t.Fatalf("ToolUseContext %+v", st.ToolUseContext)
	}
	if st.ToolUseContext.SessionID != "sess-1" || !st.ToolUseContext.Debug {
		t.Fatalf("ToolUseContext %+v", st.ToolUseContext)
	}
	if st.ToolUseContext.QuerySource != "compact_agent" {
		t.Fatalf("QuerySource %q", st.ToolUseContext.QuerySource)
	}
}

func TestLoopDriver_RunTurnLoop_maxTurns_blocksSecondAssistantAfterTools(t *testing.T) {
	turns := &SequenceTurnAssistant{Turns: []TurnResult{
		{
			Text: "need_tool",
			ToolUses: []ToolUseCall{
				{ID: "t1", Name: "bash", Input: json.RawMessage(`{}`)},
			},
		},
		{Text: "never"},
	}}
	tr := &countingToolRunner{}
	d := LoopDriver{Deps: Deps{Tools: tr, Turn: turns}, Model: "m", MaxTokens: 8}
	st := LoopState{MaxTurns: 1}
	_, _, err := d.RunTurnLoop(context.Background(), &st, "x")
	if !errors.Is(err, ErrMaxTurnsExceeded) {
		t.Fatalf("got %v", err)
	}
	if st.TurnCount != 1 {
		t.Fatalf("%+v", st)
	}
	if tr.count() != 1 {
		t.Fatalf("tools %d", tr.count())
	}
}

func TestLoopDriver_RunTurnLoopFromMessages_equivalentToRunTurnLoop(t *testing.T) {
	seed, err := InitialUserMessagesJSON("hi")
	if err != nil {
		t.Fatal(err)
	}
	mk := func() *SequenceTurnAssistant {
		return &SequenceTurnAssistant{Turns: []TurnResult{{Text: "reply"}}}
	}
	d1 := LoopDriver{Deps: Deps{Turn: mk()}, Model: "m", MaxTokens: 8}
	d2 := LoopDriver{Deps: Deps{Turn: mk()}, Model: "m", MaxTokens: 8}
	st1, st2 := LoopState{}, LoopState{}
	out1, _, err := d1.RunTurnLoop(context.Background(), &st1, "hi")
	if err != nil {
		t.Fatal(err)
	}
	out2, _, err := d2.RunTurnLoopFromMessages(context.Background(), &st2, seed)
	if err != nil {
		t.Fatal(err)
	}
	if string(out1) != string(out2) {
		t.Fatalf("transcripts differ:\n%s\nvs\n%s", out1, out2)
	}
	if st1.TurnCount != st2.TurnCount || st1.TurnCount != 1 {
		t.Fatalf("st1=%+v st2=%+v", st1, st2)
	}
}

func TestLoopDriver_RunTurnLoop_taskBudgetContext(t *testing.T) {
	inner := &SequenceTurnAssistant{Turns: []TurnResult{{Text: "x"}}}
	d := LoopDriver{
		Deps:            Deps{Turn: assertPerTurnTaskBudgetTurn{t: t, want: 777, inner: inner}},
		Model:           "m",
		MaxTokens:       8,
		TaskBudgetTotal: 777,
	}
	_, _, err := d.RunTurnLoop(context.Background(), &LoopState{}, "hi")
	if err != nil {
		t.Fatal(err)
	}
}

type assertPerTurnTaskBudgetTurn struct {
	t     *testing.T
	want  int
	inner TurnAssistant
}

func (a assertPerTurnTaskBudgetTurn) AssistantTurn(ctx context.Context, model string, maxTokens int, messagesJSON []byte) (TurnResult, error) {
	got, ok := anthropic.PerTurnTaskBudgetFromContext(ctx)
	if !ok || got != a.want {
		a.t.Fatalf("PerTurnTaskBudgetFromContext: ok=%v got=%d want=%d", ok, got, a.want)
	}
	return a.inner.AssistantTurn(ctx, model, maxTokens, messagesJSON)
}

func TestLoopDriver_RunTurnLoopFromMessages_emptySeed(t *testing.T) {
	d := LoopDriver{Deps: Deps{Turn: &SequenceTurnAssistant{}}, Model: "m", MaxTokens: 8}
	_, _, err := d.RunTurnLoopFromMessages(context.Background(), &LoopState{}, json.RawMessage(`   `))
	if err == nil {
		t.Fatal("want error")
	}
}

func TestLoopDriver_RunTurnLoop_blockingLimitBeforeAssistant(t *testing.T) {
	t.Setenv(features.EnvContextCollapse, "")
	t.Setenv(features.EnvReactiveCompact, "")
	t.Setenv(features.EnvDisableCompact, "")
	t.Setenv(features.EnvDisableAutoCompact, "")
	t.Setenv(features.EnvAutoCompact, "")
	t.Setenv(features.EnvBlockingLimitOverride, "1")
	var calls int
	seq := &SequenceTurnAssistant{Turns: []TurnResult{{Text: "ok"}}}
	d := LoopDriver{
		Deps: Deps{
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
	inner TurnAssistant
	after func()
}

func (c *countingTurnAssistant) AssistantTurn(ctx context.Context, model string, maxTokens int, messagesJSON []byte) (TurnResult, error) {
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
	seq := &SequenceTurnAssistant{Turns: []TurnResult{{Text: "ok"}}}
	d := LoopDriver{
		Deps: Deps{
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
		Deps: Deps{
			Assistant: StreamAssistantFunc(func(ctx context.Context, _ string, _ int, _ []byte) (string, error) {
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

func TestBlockingLimitPreCheckApplies_sessionMemoryFork(t *testing.T) {
	if BlockingLimitPreCheckApplies(QuerySourceSessionMemory, false) {
		t.Fatal("session_memory fork should skip blocking limit")
	}
	if BlockingLimitPreCheckApplies(QuerySourceExtractMemories, false) {
		t.Fatal("extract_memories fork should skip blocking limit")
	}
	if !BlockingLimitPreCheckApplies("", false) {
		t.Fatal("main thread should apply when gates pass")
	}
	if BlockingLimitPreCheckApplies("", true) {
		t.Fatal("post-compact continuation should skip")
	}
}

func TestCheckBlockingLimitPreAssistant_overrideLow(t *testing.T) {
	t.Setenv(features.EnvContextCollapse, "")
	t.Setenv(features.EnvReactiveCompact, "")
	t.Setenv(features.EnvDisableCompact, "")
	t.Setenv(features.EnvDisableAutoCompact, "")
	t.Setenv(features.EnvAutoCompact, "")
	t.Setenv(features.EnvBlockingLimitOverride, "1")
	long, err := InitialUserMessagesJSON(strings.Repeat("z", 800))
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckBlockingLimitPreAssistant("m", 1024, 50_000, long, 0, "", false); err == nil {
		t.Fatal("expected ErrBlockingLimit")
	} else if !errors.Is(err, ErrBlockingLimit) {
		t.Fatalf("got %v", err)
	}
}

func TestCheckBlockingLimitPreAssistant_reactiveAndAutoSkips(t *testing.T) {
	t.Setenv(features.EnvContextCollapse, "")
	t.Setenv(features.EnvReactiveCompact, "1")
	t.Setenv(features.EnvDisableCompact, "")
	t.Setenv(features.EnvDisableAutoCompact, "")
	t.Setenv(features.EnvAutoCompact, "")
	t.Setenv(features.EnvBlockingLimitOverride, "1")
	long, err := InitialUserMessagesJSON(strings.Repeat("z", 800))
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckBlockingLimitPreAssistant("m", 1024, 50_000, long, 0, "", false); err != nil {
		t.Fatalf("reactive+auto should skip synthetic blocking: %v", err)
	}
}

type stripVerifyTurnAssistant struct {
	n int
	t *testing.T
}

func (s *stripVerifyTurnAssistant) AssistantTurn(ctx context.Context, model string, maxTokens int, msgs []byte) (TurnResult, error) {
	s.n++
	if s.n == 1 {
		return TurnResult{}, anthropic.ErrPromptCacheBreakDetected
	}
	if len(msgs) == 0 || bytes.Contains(msgs, []byte("cache_control")) {
		s.t.Fatalf("call %d: expected stripped messages, got %s", s.n, msgs)
	}
	return TurnResult{Text: "ok"}, nil
}

func TestRunTurnLoop_promptCacheBreak_trimResend(t *testing.T) {
	t.Setenv(features.EnvPromptCacheBreak, "1")
	t.Setenv(features.EnvPromptCacheBreakTrimResend, "1")
	seed := json.RawMessage(`[{"role":"user","content":[{"type":"text","text":"hi","cache_control":{"type":"ephemeral"}}]}]`)
	d := LoopDriver{
		Deps: Deps{
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

func (f *failOnceCacheBreakTurn) AssistantTurn(ctx context.Context, model string, maxTokens int, msgs []byte) (TurnResult, error) {
	f.n++
	if f.n == 1 {
		return TurnResult{}, anthropic.ErrPromptCacheBreakDetected
	}
	return TurnResult{Text: "recovered"}, nil
}

func TestRunTurnLoop_promptCacheBreak_compactRecovery(t *testing.T) {
	t.Setenv(features.EnvPromptCacheBreak, "1")
	t.Setenv(features.EnvPromptCacheBreakTrimResend, "0")
	t.Setenv(features.EnvPromptCacheBreakAutoCompact, "1")
	nextTranscript := []byte(`[{"role":"user","content":[{"type":"text","text":"compact-seed"}]}]`)
	d := LoopDriver{
		Deps: Deps{
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

func (stripErrorTurnAssistant) AssistantTurn(ctx context.Context, model string, maxTokens int, msgs []byte) (TurnResult, error) {
	return TurnResult{}, anthropic.ErrPromptCacheBreakDetected
}

func TestRunTurnLoop_promptCacheBreak_stripJSONError(t *testing.T) {
	t.Setenv(features.EnvPromptCacheBreak, "1")
	t.Setenv(features.EnvPromptCacheBreakTrimResend, "1")
	seed := json.RawMessage(`not-a-json-array`)
	d := LoopDriver{
		Deps: Deps{
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

func (a *trimThenCompactTurnAssistant) AssistantTurn(ctx context.Context, model string, maxTokens int, msgs []byte) (TurnResult, error) {
	a.n++
	switch a.n {
	case 1:
		return TurnResult{}, anthropic.ErrPromptCacheBreakDetected
	case 2:
		if bytes.Contains(msgs, []byte("cache_control")) {
			a.t.Fatalf("call 2: expected stripped msgs")
		}
		return TurnResult{}, anthropic.ErrPromptCacheBreakDetected
	default:
		return TurnResult{Text: "after-compact"}, nil
	}
}

func TestRunTurnLoop_promptCacheBreak_trimThenCompact_chain(t *testing.T) {
	t.Setenv(features.EnvPromptCacheBreak, "1")
	t.Setenv(features.EnvPromptCacheBreakTrimResend, "1")
	t.Setenv(features.EnvPromptCacheBreakAutoCompact, "1")
	seed := json.RawMessage(`[{"role":"user","content":[{"type":"text","text":"x","cache_control":{"type":"ephemeral"}}]}]`)
	turn := &trimThenCompactTurnAssistant{t: t}
	d := LoopDriver{
		Deps: Deps{
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

type failTwiceCacheBreakThenOK struct {
	n int
	t *testing.T
}

func (f *failTwiceCacheBreakThenOK) AssistantTurn(ctx context.Context, model string, maxTokens int, msgs []byte) (TurnResult, error) {
	f.n++
	if f.n <= 2 {
		return TurnResult{}, anthropic.ErrPromptCacheBreakDetected
	}
	if !bytes.Contains(msgs, []byte("seed2")) {
		f.t.Fatalf("call %d: expected second compact transcript, got %s", f.n, msgs)
	}
	return TurnResult{Text: "ok-second-compact"}, nil
}

func TestRunTurnLoop_promptCacheBreak_secondCompactRound(t *testing.T) {
	t.Setenv(features.EnvPromptCacheBreak, "1")
	t.Setenv(features.EnvPromptCacheBreakTrimResend, "0")
	t.Setenv(features.EnvPromptCacheBreakAutoCompact, "1")
	var recoveryCalls int
	turn := &failTwiceCacheBreakThenOK{t: t}
	d := LoopDriver{
		Deps:  Deps{Turn: turn},
		Model: "m", MaxTokens: 8,
		PromptCacheBreakRecovery: func(ctx context.Context, msgs json.RawMessage) (json.RawMessage, bool, error) {
			recoveryCalls++
			if recoveryCalls == 1 {
				return json.RawMessage(`[{"role":"user","content":[{"type":"text","text":"seed1"}]}]`), true, nil
			}
			return json.RawMessage(`[{"role":"user","content":[{"type":"text","text":"seed2"}]}]`), true, nil
		},
	}
	st := LoopState{}
	_, text, err := d.RunTurnLoop(context.Background(), &st, "hi")
	if err != nil {
		t.Fatal(err)
	}
	if text != "ok-second-compact" {
		t.Fatalf("text %q", text)
	}
	if recoveryCalls != 2 {
		t.Fatalf("recovery calls: want 2, got %d", recoveryCalls)
	}
	if turn.n != 3 {
		t.Fatalf("AssistantTurn calls: want 3 (break, break after seed1, ok after seed2), got %d", turn.n)
	}
}

func TestLoopDriver_SnipTokensFreedAccum_onHistorySnip(t *testing.T) {
	t.Setenv(features.EnvPromptCacheBreak, "")
	t.Setenv(features.EnvPromptCacheBreakTrimResend, "0")
	t.Setenv(features.EnvPromptCacheBreakAutoCompact, "")
	var parts []string
	for i := 0; i < 6; i++ {
		parts = append(parts, `{"role":"user","content":[{"type":"text","text":"`+strings.Repeat("W", 120)+`"}]}`)
	}
	seed := json.RawMessage("[" + strings.Join(parts, ",") + "]")
	// assistantTurnWithPromptCacheBreakHandling may call AssistantTurn more than once when cache-break recovery is active in the environment.
	turns := &SequenceTurnAssistant{Turns: []TurnResult{
		{Text: "done"}, {Text: "done"}, {Text: "done"},
	}}
	d := LoopDriver{
		Deps:                 Deps{Turn: turns},
		Model:                "m",
		MaxTokens:            8,
		HistorySnipMaxBytes:  500,
		HistorySnipMaxRounds: 20,
		ContextWindowTokens:  500_000,
	}
	st := LoopState{}
	_, _, err := d.RunTurnLoopFromMessages(context.Background(), &st, seed)
	if err != nil {
		t.Fatal(err)
	}
	if st.SnipTokensFreedAccum <= 0 {
		t.Fatalf("expected SnipTokensFreedAccum > 0 after snip, got %d", st.SnipTokensFreedAccum)
	}
	firstAccum := st.SnipTokensFreedAccum
	st2 := LoopState{}
	_, _, err = d.RunTurnLoopFromMessages(context.Background(), &st2, seed)
	if err != nil {
		t.Fatal(err)
	}
	if st2.SnipTokensFreedAccum <= 0 {
		t.Fatalf("expected fresh RunTurnLoop to accumulate snip again, got %d", st2.SnipTokensFreedAccum)
	}
	if st2.SnipTokensFreedAccum != firstAccum {
		t.Fatalf("deterministic snip accum: want %d, got %d", firstAccum, st2.SnipTokensFreedAccum)
	}
}
