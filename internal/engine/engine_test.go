package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/2456868764/rabbit-code/internal/anthropic"
	"github.com/2456868764/rabbit-code/internal/compact"
	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/query"
	"github.com/2456868764/rabbit-code/internal/querydeps"
)

func drainChFor(d time.Duration, ch <-chan EngineEvent) {
	deadline := time.After(d)
	for {
		select {
		case <-deadline:
			return
		case <-ch:
		}
	}
}

func TestEngine_CompactE2E_longTranscriptTriggersExecutor(t *testing.T) {
	longReply := strings.Repeat("xy ", 400)
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return longReply, nil
			}),
		},
		Model: "m", MaxTokens: 64,
		CompactAdvisor: func(_ query.LoopState, transcriptJSON []byte) (bool, bool) {
			return len(transcriptJSON) > 1200, false
		},
		CompactExecutor: func(_ context.Context, phase compact.RunPhase, _ []byte) (string, error) {
			_ = phase
			return "e2e-compact-summary", nil
		},
	})
	e.Submit("hi")
	ch := e.Events()
	deadline := time.After(3 * time.Second)
	var sawSuggest, sawResult, sawDone bool
	for {
		select {
		case ev := <-ch:
			switch ev.Kind {
			case EventKindCompactSuggest:
				sawSuggest = true
			case EventKindCompactResult:
				sawResult = true
				if ev.CompactSummary != "e2e-compact-summary" || ev.Err != nil {
					t.Fatalf("%+v", ev)
				}
			case EventKindDone:
				sawDone = true
				goto compactE2EDone
			case EventKindError:
				t.Fatal(ev.Err)
			}
		case <-deadline:
			t.Fatalf("timeout suggest=%v result=%v done=%v", sawSuggest, sawResult, sawDone)
		}
	}
compactE2EDone:
	e.Wait()
	if !sawSuggest || !sawResult {
		t.Fatalf("suggest=%v result=%v", sawSuggest, sawResult)
	}
}

func TestEngine_TokenBudget_blocksOversizeResolvedText(t *testing.T) {
	t.Setenv(features.EnvTokenBudget, "true")
	t.Setenv(features.EnvTokenBudgetMaxInputBytes, "10")
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "ok", nil
			}),
		},
	})
	e.Submit("12345678901")
	var saw error
	for {
		select {
		case ev := <-e.Events():
			if ev.Kind == EventKindError {
				saw = ev.Err
				goto budgetDone
			}
			if ev.Kind == EventKindAssistantText {
				t.Fatal("assistant should not run")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
	}
budgetDone:
	e.Wait()
	if !errors.Is(saw, ErrTokenBudgetExceeded) {
		t.Fatalf("got %v", saw)
	}
}

func TestEngine_TokenBudget_blocksByTokenEstimate(t *testing.T) {
	t.Setenv(features.EnvTokenBudget, "true")
	t.Setenv(features.EnvTokenBudgetMaxInputBytes, "999999")
	t.Setenv(features.EnvTokenBudgetMaxInputTokens, "1")
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "ok", nil
			}),
		},
	})
	e.Submit("abcde")
	var saw error
	for {
		select {
		case ev := <-e.Events():
			if ev.Kind == EventKindError {
				saw = ev.Err
				goto tokDone
			}
			if ev.Kind == EventKindAssistantText {
				t.Fatal("assistant should not run")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
	}
tokDone:
	e.Wait()
	if !errors.Is(saw, ErrTokenBudgetExceeded) {
		t.Fatalf("got %v", saw)
	}
}

func TestEngine_TokenBudget_blocksOversizeMemdirInject(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "big.txt")
	if err := os.WriteFile(p, []byte("12345678901"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(features.EnvTokenBudget, "true")
	t.Setenv(features.EnvTokenBudgetMaxAttachmentBytes, "5")
	e := New(context.Background(), &Config{
		MemdirPaths: []string{p},
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "ok", nil
			}),
		},
	})
	e.Submit("x")
	var saw error
	for {
		select {
		case ev := <-e.Events():
			if ev.Kind == EventKindError {
				saw = ev.Err
				goto attDone
			}
			if ev.Kind == EventKindAssistantText {
				t.Fatal("assistant should not run")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
	}
attDone:
	e.Wait()
	if !errors.Is(saw, ErrTokenBudgetExceeded) {
		t.Fatalf("got %v", saw)
	}
}

func TestEngine_Submit_withStreamAssistant(t *testing.T) {
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(ctx context.Context, model string, maxTokens int, messagesJSON []byte) (string, error) {
				if model != "m" || maxTokens != 16 {
					t.Fatalf("model=%q max=%d", model, maxTokens)
				}
				return "assistant-out", nil
			}),
		},
		Model:     "m",
		MaxTokens: 16,
	})
	e.Submit("user-in")
	var kinds []EventKind
	var lastAssist string
	for i := 0; i < 3; i++ {
		select {
		case ev := <-e.Events():
			kinds = append(kinds, ev.Kind)
			if ev.Kind == EventKindAssistantText {
				lastAssist = ev.AssistText
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout at %d: %v", len(kinds), kinds)
		}
	}
	e.Wait()
	if kinds[0] != EventKindUserSubmit || kinds[1] != EventKindAssistantText || kinds[2] != EventKindDone {
		t.Fatalf("got %v", kinds)
	}
	if lastAssist != "assistant-out" {
		t.Fatalf("assist %q", lastAssist)
	}
}

func TestEngine_Submit_streamAssistantError(t *testing.T) {
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "", errors.New("stream err")
			}),
		},
	})
	e.Submit("x")
	var kinds []EventKind
	for i := 0; i < 2; i++ {
		ev := <-e.Events()
		kinds = append(kinds, ev.Kind)
	}
	e.Wait()
	if kinds[0] != EventKindUserSubmit || kinds[1] != EventKindError {
		t.Fatalf("got %v", kinds)
	}
}

func TestEngine_Submit_emitsSequence(t *testing.T) {
	e := NewEngine(context.Background())
	e.Submit("hello")
	var kinds []EventKind
	for i := 0; i < 3; i++ {
		select {
		case ev := <-e.Events():
			kinds = append(kinds, ev.Kind)
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout after %d events: %v", len(kinds), kinds)
		}
	}
	e.Wait()
	if kinds[0] != EventKindUserSubmit || kinds[1] != EventKindAssistantText || kinds[2] != EventKindDone {
		t.Fatalf("got %v", kinds)
	}
}

func TestEngine_RunTurnLoop_toolEvents(t *testing.T) {
	tr := &countingToolRunner{}
	turns := &querydeps.SequenceTurnAssistant{Turns: []querydeps.TurnResult{
		{Text: "a", ToolUses: []querydeps.ToolUseCall{{ID: "1", Name: "bash", Input: json.RawMessage(`{}`)}}},
		{Text: "b"},
	}}
	e := New(context.Background(), &Config{
		Deps:  querydeps.Deps{Tools: tr, Turn: turns},
		Model: "m", MaxTokens: 8,
	})
	e.Submit("hi")
	var kinds []EventKind
	for {
		select {
		case ev := <-e.Events():
			kinds = append(kinds, ev.Kind)
			if ev.Kind == EventKindDone {
				goto toolDone
			}
		case <-time.After(2 * time.Second):
			t.Fatalf("timeout kinds=%v", kinds)
		}
	}
toolDone:
	e.Wait()
	if kinds[0] != EventKindUserSubmit {
		t.Fatalf("got %v", kinds)
	}
	if tr.n != 1 {
		t.Fatalf("tool runs %d", tr.n)
	}
	if kinds[len(kinds)-1] != EventKindDone {
		t.Fatalf("last %v", kinds[len(kinds)-1])
	}
}

func TestEngine_MemdirInject_prependsFragments(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "m.txt")
	if err := os.WriteFile(p, []byte("fragment-line"), 0o644); err != nil {
		t.Fatal(err)
	}
	var sawMemdir bool
	var lastUser string
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(_ context.Context, _ string, _ int, messagesJSON []byte) (string, error) {
				lastUser = string(messagesJSON)
				return "ok", nil
			}),
		},
		MemdirPaths: []string{p},
	})
	e.Submit("user")
	for {
		select {
		case ev := <-e.Events():
			if ev.Kind == EventKindMemdirInject {
				sawMemdir = true
				if ev.MemdirFragmentCount != 1 {
					t.Fatalf("count %d", ev.MemdirFragmentCount)
				}
			}
			if ev.Kind == EventKindDone {
				goto done
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
	}
done:
	e.Wait()
	if !sawMemdir {
		t.Fatal("no memdir event")
	}
	if lastUser == "" || !strings.Contains(lastUser, "fragment-line") || !strings.Contains(lastUser, "user") {
		t.Fatalf("messages %q", lastUser)
	}
}

func TestEngine_CompactSuggest_afterLoop(t *testing.T) {
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "x", nil
			}),
		},
		CompactAdvisor: func(_ query.LoopState, _ []byte) (bool, bool) {
			return true, false
		},
	})
	e.Submit("u")
	var sawCompact bool
	for {
		ev := <-e.Events()
		if ev.Kind == EventKindCompactSuggest {
			sawCompact = true
			if ev.CompactPhase != "auto_pending" || !ev.SuggestAutoCompact {
				t.Fatalf("%+v", ev)
			}
		}
		if ev.Kind == EventKindDone {
			break
		}
	}
	e.Wait()
	if !sawCompact {
		t.Fatal("expected compact suggest")
	}
}

func TestEngine_Error_anthropicKindAndRecoverable(t *testing.T) {
	apiErr := &anthropic.APIError{Kind: anthropic.KindPromptTooLong, Status: 400, Msg: "ptl"}
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "", fmt.Errorf("wrap: %w", apiErr)
			}),
		},
	})
	e.Submit("x")
	var sawErr *EngineEvent
	for {
		ev := <-e.Events()
		if ev.Kind == EventKindError {
			sawErr = &ev
			break
		}
	}
	e.Wait()
	if sawErr == nil {
		t.Fatal("no error event")
	}
	if sawErr.APIErrorKind != string(anthropic.KindPromptTooLong) || !sawErr.RecoverableCompact {
		t.Fatalf("%+v", sawErr)
	}
}

func TestEngine_StopHooks_order(t *testing.T) {
	var order []int
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "ok", nil
			}),
		},
		StopHooks: []StopHookFunc{
			func(context.Context, query.LoopState, error) { order = append(order, 1) },
			func(context.Context, query.LoopState, error) { order = append(order, 2) },
		},
		StopHook: func(context.Context, query.LoopState, error) { order = append(order, 3) },
	})
	e.Submit("x")
	drainUntilTerminal(t, e.Events())
	e.Wait()
	if len(order) != 3 || order[0] != 1 || order[1] != 2 || order[2] != 3 {
		t.Fatalf("got %v", order)
	}
}

func TestEngine_StopHook_successAndFailure(t *testing.T) {
	var calls int
	var lastErr error
	hook := func(_ context.Context, _ query.LoopState, err error) {
		calls++
		lastErr = err
	}

	e1 := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "ok", nil
			}),
		},
		StopHook: hook,
	})
	e1.Submit("a")
	drainUntilTerminal(t, e1.Events())
	e1.Wait()
	if calls != 1 || lastErr != nil {
		t.Fatalf("calls=%d err=%v", calls, lastErr)
	}

	e2 := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "", errors.New("fail")
			}),
		},
		StopHook: hook,
	})
	e2.Submit("b")
	drainUntilTerminal(t, e2.Events())
	e2.Wait()
	if calls != 2 || lastErr == nil {
		t.Fatalf("calls=%d err=%v", calls, lastErr)
	}
}

func TestEngine_MaxAssistantTurns(t *testing.T) {
	// One assistant message with tools consumes a turn; a second assistant round must not start when MaxTurns==1.
	seq := &querydeps.SequenceTurnAssistant{Turns: []querydeps.TurnResult{
		{Text: "t1", ToolUses: []querydeps.ToolUseCall{{ID: "x", Name: "bash", Input: json.RawMessage(`{}`)}}},
		{Text: "t2"},
	}}
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Turn:  seq,
			Tools: &countingToolRunner{},
		},
		MaxAssistantTurns: 1,
	})
	e.Submit("x")
	var sawErr error
	for {
		ev := <-e.Events()
		if ev.Kind == EventKindError {
			sawErr = ev.Err
			break
		}
	}
	e.Wait()
	if !errors.Is(sawErr, query.ErrMaxTurnsExceeded) {
		t.Fatalf("got %v", sawErr)
	}
}

func TestEngine_RecoverableError_emitsCompactSuggestWhenConfigured(t *testing.T) {
	apiErr := &anthropic.APIError{Kind: anthropic.KindPromptTooLong, Status: 400, Msg: "ptl"}
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "", fmt.Errorf("w: %w", apiErr)
			}),
		},
		SuggestCompactOnRecoverableError: true,
	})
	e.Submit("x")
	var kinds []EventKind
	for {
		ev := <-e.Events()
		kinds = append(kinds, ev.Kind)
		if ev.Kind == EventKindError {
			break
		}
	}
	e.Wait()
	if len(kinds) < 3 || kinds[0] != EventKindUserSubmit || kinds[1] != EventKindCompactSuggest || kinds[2] != EventKindError {
		t.Fatalf("got %v", kinds)
	}
}

func TestEngine_OrphanPermission_advisor(t *testing.T) {
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "y", nil
			}),
		},
		OrphanPermissionAdvisor: func(_ query.LoopState) (string, bool) {
			return "toolu_orphan_1", true
		},
	})
	e.Submit("z")
	var saw bool
	for {
		ev := <-e.Events()
		if ev.Kind == EventKindOrphanPermission && ev.OrphanToolUseID == "toolu_orphan_1" {
			saw = true
		}
		if ev.Kind == EventKindDone {
			break
		}
	}
	e.Wait()
	if !saw {
		t.Fatal("expected orphan permission event")
	}
}

func drainUntilTerminal(t *testing.T, ch <-chan EngineEvent) {
	t.Helper()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev := <-ch:
			if ev.Kind == EventKindDone || ev.Kind == EventKindError {
				return
			}
		case <-deadline:
			t.Fatal("timeout waiting for terminal event")
		}
	}
}

type countingToolRunner struct{ n int }

func (c *countingToolRunner) RunTool(context.Context, string, []byte) ([]byte, error) {
	c.n++
	return []byte(`{}`), nil
}

func TestEngine_RecoverStrategy_secondAttemptSucceeds(t *testing.T) {
	var n int
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				n++
				if n == 1 {
					return "", errors.New("transient")
				}
				return "ok", nil
			}),
		},
		RecoverStrategy: func(context.Context, query.LoopState, error) bool { return true },
	})
	e.Submit("x")
	ch := e.Events()
	deadline := time.After(2 * time.Second)
	var sawDone bool
	for {
		select {
		case ev := <-ch:
			if ev.Kind == EventKindDone {
				sawDone = true
				if ev.LoopTurnCount != 1 {
					t.Fatalf("turns %d", ev.LoopTurnCount)
				}
				goto recDone
			}
			if ev.Kind == EventKindError {
				t.Fatalf("unexpected error before retry success: %v", ev.Err)
			}
		case <-deadline:
			t.Fatal("timeout")
		}
	}
recDone:
	e.Wait()
	if !sawDone || n != 2 {
		t.Fatalf("done=%v n=%d", sawDone, n)
	}
}

func TestEngine_RecoverStrategy_StopHook_seesSubmitRecoverContinue(t *testing.T) {
	var captured query.LoopState
	var n int
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				n++
				if n == 1 {
					return "", errors.New("transient")
				}
				return "ok", nil
			}),
		},
		RecoverStrategy: func(context.Context, query.LoopState, error) bool { return true },
		StopHooks: []StopHookFunc{
			func(_ context.Context, st query.LoopState, _ error) { captured = st },
		},
	})
	e.Submit("x")
	drainUntilTerminal(t, e.Events())
	e.Wait()
	if captured.LoopContinue.Reason != query.ContinueReasonSubmitRecoverRetry {
		t.Fatalf("want submit_recover_retry, got %+v", captured.LoopContinue)
	}
}

func TestEngine_RecoverStrategy_maxOutputTokens_recordsRecoveryContinue(t *testing.T) {
	var captured query.LoopState
	var n int
	apiErr := &anthropic.APIError{Kind: anthropic.KindMaxOutputTokens, Status: 400, Msg: "otk"}
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				n++
				if n == 1 {
					return "", fmt.Errorf("wrap: %w", apiErr)
				}
				return "ok", nil
			}),
		},
		RecoverStrategy: func(context.Context, query.LoopState, error) bool { return true },
		StopHooks: []StopHookFunc{
			func(_ context.Context, st query.LoopState, _ error) { captured = st },
		},
	})
	e.Submit("x")
	drainUntilTerminal(t, e.Events())
	e.Wait()
	if captured.LoopContinue.Reason != query.ContinueReasonMaxOutputTokensRecovery || captured.LoopContinue.Attempt != 1 {
		t.Fatalf("got %+v", captured.LoopContinue)
	}
	if captured.MaxOutputTokensRecoveryCount != 1 {
		t.Fatalf("count %d", captured.MaxOutputTokensRecoveryCount)
	}
}

func TestEngine_recoverableError_compactStub_recordsReactiveContinue(t *testing.T) {
	var captured query.LoopState
	apiErr := &anthropic.APIError{Kind: anthropic.KindPromptTooLong, Status: 400, Msg: "ptl"}
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "", fmt.Errorf("w: %w", apiErr)
			}),
		},
		SuggestCompactOnRecoverableError: true,
		CompactExecutor:                  compact.ExecuteStub,
		StopHooks: []StopHookFunc{
			func(_ context.Context, st query.LoopState, _ error) { captured = st },
		},
	})
	e.Submit("x")
	drainUntilTerminal(t, e.Events())
	e.Wait()
	if captured.LoopContinue.Reason != query.ContinueReasonReactiveCompactRetry {
		t.Fatalf("got %+v", captured.LoopContinue)
	}
}

func TestEngine_StopHookBlockingContinue_runsSecondTurnLoop(t *testing.T) {
	turns := &querydeps.SequenceTurnAssistant{Turns: []querydeps.TurnResult{
		{Text: "a"},
		{Text: "b"},
	}}
	var nCont int
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{Turn: turns},
		Model: "m", MaxTokens: 8,
		StopHookBlockingContinue: func(context.Context, query.LoopState) bool {
			nCont++
			return nCont == 1
		},
	})
	e.Submit("hi")
	ch := e.Events()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev := <-ch:
			if ev.Kind == EventKindDone {
				if ev.LoopTurnCount != 2 {
					t.Fatalf("want 2 turn counts across continuations, got %d", ev.LoopTurnCount)
				}
				if nCont != 2 {
					t.Fatalf("StopHookBlockingContinue calls %d", nCont)
				}
				e.Wait()
				return
			}
		case <-deadline:
			t.Fatal("timeout")
		}
	}
}

func TestEngine_TokenBudgetContinueAfterTurn_secondLoop(t *testing.T) {
	t.Setenv(features.EnvTokenBudget, "true")
	t.Setenv(features.EnvTokenBudgetMaxInputBytes, "999999")
	turns := &querydeps.SequenceTurnAssistant{Turns: []querydeps.TurnResult{
		{Text: "t1"},
		{Text: "t2"},
	}}
	var nTok int
	var captured query.LoopState
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{Turn: turns},
		Model: "m", MaxTokens: 8,
		TokenBudgetContinueAfterTurn: func(context.Context, query.LoopState, json.RawMessage) bool {
			nTok++
			return nTok == 1
		},
		StopHooks: []StopHookFunc{
			func(_ context.Context, st query.LoopState, _ error) { captured = st },
		},
	})
	e.Submit("hi")
	drainUntilTerminal(t, e.Events())
	e.Wait()
	if captured.LoopContinue.Reason != query.ContinueReasonTokenBudgetContinuation {
		t.Fatalf("want token_budget_continuation, got %+v", captured.LoopContinue)
	}
	if nTok != 2 {
		t.Fatalf("TokenBudgetContinueAfterTurn calls %d", nTok)
	}
}

func TestEngine_ContextCollapseDrain_recordsContinueOnPTL(t *testing.T) {
	t.Setenv(features.EnvContextCollapse, "true")
	var captured query.LoopState
	apiErr := &anthropic.APIError{Kind: anthropic.KindPromptTooLong, Status: 400, Msg: "ptl"}
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "", fmt.Errorf("w: %w", apiErr)
			}),
		},
		ContextCollapseDrain: func(_ context.Context, _ *query.LoopState, _ json.RawMessage) (json.RawMessage, int, bool) {
			return json.RawMessage(`[]`), 2, true
		},
		StopHooks: []StopHookFunc{
			func(_ context.Context, st query.LoopState, _ error) { captured = st },
		},
	})
	e.Submit("x")
	drainUntilTerminal(t, e.Events())
	e.Wait()
	if captured.LoopContinue.Reason != query.ContinueReasonCollapseDrainRetry || captured.LoopContinue.Committed != 2 {
		t.Fatalf("got %+v", captured.LoopContinue)
	}
}

func TestEngine_ContextCollapseDrain_recoverRetryUsesDrainedSeed(t *testing.T) {
	t.Setenv(features.EnvContextCollapse, "true")
	apiErr := &anthropic.APIError{Kind: anthropic.KindPromptTooLong, Status: 400, Msg: "ptl"}
	trimmed, err := query.InitialUserMessagesJSON("SEED_MARKER")
	if err != nil {
		t.Fatal(err)
	}
	var n int
	var secondBody string
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(_ context.Context, _ string, _ int, messagesJSON []byte) (string, error) {
				n++
				if n == 1 {
					return "", fmt.Errorf("w: %w", apiErr)
				}
				secondBody = string(messagesJSON)
				return "ok", nil
			}),
		},
		RecoverStrategy: func(context.Context, query.LoopState, error) bool { return true },
		ContextCollapseDrain: func(_ context.Context, _ *query.LoopState, _ json.RawMessage) (json.RawMessage, int, bool) {
			return trimmed, 1, true
		},
	})
	e.Submit("original-user-should-not-appear-on-retry")
	drainUntilTerminal(t, e.Events())
	e.Wait()
	if n != 2 {
		t.Fatalf("want 2 assistant calls, got %d", n)
	}
	if !strings.Contains(secondBody, "SEED_MARKER") || strings.Contains(secondBody, "original-user-should-not-appear-on-retry") {
		t.Fatalf("retry should use drained transcript; got %q", secondBody)
	}
}

func TestEngine_ConfigAgentIDNonInteractive_inLoopState(t *testing.T) {
	var captured query.LoopState
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "x", nil
			}),
		},
		Model:          "custom-model",
		AgentID:        "engine-agent",
		NonInteractive: true,
		StopHooks: []StopHookFunc{
			func(_ context.Context, st query.LoopState, _ error) { captured = st },
		},
	})
	e.Submit("hi")
	drainUntilTerminal(t, e.Events())
	e.Wait()
	if captured.ToolUseContext.AgentID != "engine-agent" || !captured.ToolUseContext.NonInteractive {
		t.Fatalf("%+v", captured.ToolUseContext)
	}
	if captured.ToolUseContext.MainLoopModel != "custom-model" {
		t.Fatalf("model %q", captured.ToolUseContext.MainLoopModel)
	}
	if len(captured.MessagesJSON) == 0 || !strings.Contains(string(captured.MessagesJSON), "hi") {
		t.Fatalf("MessagesJSON %s", captured.MessagesJSON)
	}
}

func TestEngine_BashStubToolRunner(t *testing.T) {
	turns := &querydeps.SequenceTurnAssistant{Turns: []querydeps.TurnResult{
		{Text: "run", ToolUses: []querydeps.ToolUseCall{{ID: "t1", Name: "bash", Input: json.RawMessage(`{"cmd":"ls"}`)}}},
		{Text: "done"},
	}}
	e := New(context.Background(), &Config{
		Deps:  querydeps.Deps{Tools: querydeps.BashStubToolRunner{}, Turn: turns},
		Model: "m", MaxTokens: 8,
	})
	e.Submit("hi")
	drainUntilTerminal(t, e.Events())
	e.Wait()
}

func TestEngine_ToolCallFailed_emitsOrphanFromError(t *testing.T) {
	turns := &querydeps.SequenceTurnAssistant{Turns: []querydeps.TurnResult{
		{Text: "x", ToolUses: []querydeps.ToolUseCall{{ID: "orph1", Name: "bash", Input: json.RawMessage(`{}`)}}},
	}}
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Turn: turns,
			Tools: toolRunnerFunc(func(_ context.Context, _ string, _ []byte) ([]byte, error) {
				return nil, &querydeps.OrphanPermissionError{ToolUseID: "orph1"}
			}),
		},
		Model: "m", MaxTokens: 8,
	})
	e.Submit("z")
	ch := e.Events()
	deadline := time.After(2 * time.Second)
	var sawFail, sawOrphan bool
	for {
		select {
		case ev := <-ch:
			if ev.Kind == EventKindToolCallFailed {
				sawFail = true
			}
			if ev.Kind == EventKindOrphanPermission && ev.OrphanToolUseID == "orph1" {
				sawOrphan = true
			}
			if ev.Kind == EventKindError {
				if !sawFail || !sawOrphan {
					t.Fatalf("fail=%v orphan=%v", sawFail, sawOrphan)
				}
				e.Wait()
				return
			}
		case <-deadline:
			t.Fatal("timeout")
		}
	}
}

type toolRunnerFunc func(context.Context, string, []byte) ([]byte, error)

func (f toolRunnerFunc) RunTool(ctx context.Context, name string, in []byte) ([]byte, error) {
	return f(ctx, name, in)
}

func TestEngine_CompactExecutor_afterAdvisor(t *testing.T) {
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "y", nil
			}),
		},
		CompactAdvisor: func(_ query.LoopState, _ []byte) (bool, bool) { return true, false },
		CompactExecutor: func(_ context.Context, phase compact.RunPhase, transcript []byte) (string, error) {
			_ = phase
			if len(transcript) == 0 {
				return "", errors.New("empty transcript")
			}
			return "summary-ok", nil
		},
	})
	e.Submit("u")
	ch := e.Events()
	deadline := time.After(2 * time.Second)
	var sawSuggest, sawResult bool
	for {
		select {
		case ev := <-ch:
			switch ev.Kind {
			case EventKindCompactSuggest:
				sawSuggest = true
			case EventKindCompactResult:
				sawResult = true
				if ev.CompactSummary != "summary-ok" || ev.Err != nil {
					t.Fatalf("%+v", ev)
				}
			case EventKindDone:
				e.Wait()
				if !sawSuggest || !sawResult {
					t.Fatalf("suggest=%v result=%v", sawSuggest, sawResult)
				}
				return
			case EventKindError:
				t.Fatal(ev.Err)
			}
		case <-deadline:
			t.Fatal("timeout")
		}
	}
}

func TestEngine_Submit_withStreamAssistant_doneTurnCount(t *testing.T) {
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "out", nil
			}),
		},
		Model: "m", MaxTokens: 16,
	})
	e.Submit("in")
	ch := e.Events()
	deadline := time.After(2 * time.Second)
	for {
		select {
		case ev := <-ch:
			if ev.Kind == EventKindDone {
				if ev.LoopTurnCount != 1 {
					t.Fatalf("turns %d", ev.LoopTurnCount)
				}
				e.Wait()
				return
			}
		case <-deadline:
			t.Fatal("timeout")
		}
	}
}

func TestEngine_Phase5_breakCacheTemplatesMicrocompactEvents(t *testing.T) {
	t.Setenv(features.EnvBreakCacheCommand, "true")
	t.Setenv(features.EnvTemplates, "true")
	t.Setenv(features.EnvTemplateNames, "a,b")
	t.Setenv(features.EnvCachedMicrocompact, "true")
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "x", nil
			}),
		},
	})
	e.Submit("u")
	ch := e.Events()
	deadline := time.After(2 * time.Second)
	var breakSeen, tplSeen, microSeen bool
	for {
		select {
		case ev := <-ch:
			switch ev.Kind {
			case EventKindBreakCacheCommand:
				breakSeen = true
			case EventKindTemplatesActive:
				tplSeen = true
				if ev.PhaseDetail != "a,b" {
					t.Fatalf("%q", ev.PhaseDetail)
				}
			case EventKindCachedMicrocompactActive:
				microSeen = true
			case EventKindDone:
				if !breakSeen || !tplSeen || !microSeen {
					t.Fatalf("break=%v tpl=%v micro=%v", breakSeen, tplSeen, microSeen)
				}
				e.Wait()
				return
			}
		case <-deadline:
			t.Fatal("timeout")
		}
	}
}

func TestEngine_templateMarkdownAppendixFromDir(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("BODY"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(features.EnvTemplates, "true")
	t.Setenv(features.EnvTemplateNames, "a")
	var captured string
	e := New(context.Background(), &Config{
		TemplateDir: dir,
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(_ context.Context, _ string, _ int, messagesJSON []byte) (string, error) {
				captured = string(messagesJSON)
				return "x", nil
			}),
		},
	})
	e.Submit("u")
	for {
		select {
		case ev := <-e.Events():
			if ev.Kind == EventKindDone {
				if captured == "" || !strings.Contains(captured, "BODY") || !strings.Contains(captured, "## Template a") {
					t.Fatalf("messages %q", captured)
				}
				e.Wait()
				return
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestEngine_UserSubmit_carriesPhase5ModeTags(t *testing.T) {
	t.Setenv(features.EnvUltrathink, "true")
	t.Setenv(features.EnvUltraplan, "true")
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "x", nil
			}),
		},
	})
	e.Submit("hi")
	ev := <-e.Events()
	if ev.Kind != EventKindUserSubmit || ev.PhaseDetail != "ultrathink,ultraplan" {
		t.Fatalf("%+v", ev)
	}
	for {
		select {
		case ev2 := <-e.Events():
			if ev2.Kind == EventKindDone {
				e.Wait()
				return
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestEngine_Phase5_ultrathinkInjectsIntoMessagesJSON(t *testing.T) {
	t.Setenv(features.EnvUltrathink, "true")
	var captured string
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(_ context.Context, _ string, _ int, messagesJSON []byte) (string, error) {
				captured = string(messagesJSON)
				return "ok", nil
			}),
		},
	})
	e.Submit("plain")
	ch := e.Events()
	for {
		select {
		case ev := <-ch:
			if ev.Kind == EventKindDone {
				if captured == "" || !strings.Contains(captured, "ULTRATHINK") {
					t.Fatalf("messages %q", captured)
				}
				e.Wait()
				return
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestEngine_Phase5_reactiveCompactFromEnv_minTokens(t *testing.T) {
	t.Setenv(features.EnvReactiveCompact, "true")
	t.Setenv(features.EnvReactiveCompactMinBytes, "999999")
	t.Setenv(features.EnvReactiveCompactMinTokens, "1")
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "r", nil
			}),
		},
		CompactExecutor: compact.ExecuteStub,
	})
	e.Submit("u")
	ch := e.Events()
	var sawReactive bool
	for {
		select {
		case ev := <-ch:
			if ev.Kind == EventKindCompactSuggest && ev.SuggestReactiveCompact {
				sawReactive = true
			}
			if ev.Kind == EventKindDone {
				if !sawReactive {
					t.Fatal("expected reactive compact suggest (token threshold)")
				}
				e.Wait()
				return
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestEngine_Phase5_reactiveCompactFromEnv(t *testing.T) {
	t.Setenv(features.EnvReactiveCompact, "true")
	t.Setenv(features.EnvReactiveCompactMinBytes, "1")
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "r", nil
			}),
		},
		CompactExecutor: compact.ExecuteStub,
	})
	e.Submit("u")
	ch := e.Events()
	var sawReactive bool
	for {
		select {
		case ev := <-ch:
			if ev.Kind == EventKindCompactSuggest && ev.SuggestReactiveCompact {
				sawReactive = true
			}
			if ev.Kind == EventKindDone {
				if !sawReactive {
					t.Fatal("expected reactive compact suggest")
				}
				e.Wait()
				return
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestEngine_Phase5_historySnipBetweenRounds(t *testing.T) {
	t.Setenv(features.EnvHistorySnip, "true")
	t.Setenv(features.EnvHistorySnipMaxBytes, "280")
	t.Setenv(features.EnvHistorySnipMaxRounds, "2")
	long := strings.Repeat("z", 350)
	seq := &querydeps.SequenceTurnAssistant{Turns: []querydeps.TurnResult{
		{Text: long, ToolUses: []querydeps.ToolUseCall{{ID: "1", Name: "bash", Input: json.RawMessage(`{}`)}}},
		{Text: "end"},
	}}
	e := New(context.Background(), &Config{
		Deps:  querydeps.Deps{Turn: seq, Tools: querydeps.BashStubToolRunner{}},
		Model: "m", MaxTokens: 8,
	})
	e.Submit("hi")
	ch := e.Events()
	var sawSnip bool
	for {
		select {
		case ev := <-ch:
			if ev.Kind == EventKindHistorySnipApplied {
				sawSnip = true
			}
			if ev.Kind == EventKindDone {
				if !sawSnip {
					t.Fatal("expected history snip event")
				}
				e.Wait()
				return
			}
		case <-time.After(3 * time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestEngine_SnipCompactBetweenRounds(t *testing.T) {
	t.Setenv(features.EnvSnipCompact, "true")
	t.Setenv(features.EnvSnipCompactMaxBytes, "280")
	t.Setenv(features.EnvSnipCompactMaxRounds, "2")
	long := strings.Repeat("s", 350)
	seq := &querydeps.SequenceTurnAssistant{Turns: []querydeps.TurnResult{
		{Text: long, ToolUses: []querydeps.ToolUseCall{{ID: "1", Name: "bash", Input: json.RawMessage(`{}`)}}},
		{Text: "end"},
	}}
	e := New(context.Background(), &Config{
		Deps:  querydeps.Deps{Turn: seq, Tools: querydeps.BashStubToolRunner{}},
		Model: "m", MaxTokens: 8,
	})
	e.Submit("hi")
	ch := e.Events()
	var saw bool
	for {
		select {
		case ev := <-ch:
			if ev.Kind == EventKindSnipCompactApplied {
				saw = true
			}
			if ev.Kind == EventKindDone {
				if !saw {
					t.Fatal("expected snip compact event")
				}
				e.Wait()
				return
			}
		case <-time.After(3 * time.Second):
			t.Fatal("timeout")
		}
	}
}

type cacheBreakFireTurn struct{}

func (cacheBreakFireTurn) AssistantTurn(ctx context.Context, model string, maxTokens int, msgs []byte) (querydeps.TurnResult, error) {
	_ = model
	_ = maxTokens
	_ = msgs
	if cb, ok := querydeps.OnPromptCacheBreakFromContext(ctx); ok && cb != nil {
		cb()
	}
	return querydeps.TurnResult{Text: "ok"}, nil
}

func TestEngine_promptCacheBreakSuggestCompact(t *testing.T) {
	t.Setenv(features.EnvPromptCacheBreak, "")
	t.Setenv(features.EnvPromptCacheBreakSuggestCompact, "true")
	var sawSuggest bool
	e := New(context.Background(), &Config{
		Deps: querydeps.Deps{
			Turn: cacheBreakFireTurn{},
		},
		CompactExecutor: compact.ExecuteStub,
	})
	e.Submit("hi")
	for {
		select {
		case ev := <-e.Events():
			if ev.Kind == EventKindCompactSuggest && ev.SuggestReactiveCompact {
				sawSuggest = true
			}
			if ev.Kind == EventKindDone {
				if !sawSuggest {
					t.Fatal("expected reactive compact after cache break hook")
				}
				e.Wait()
				return
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestEngine_SubmitCancelRace(t *testing.T) {
	for i := 0; i < 40; i++ {
		e := NewEngine(context.Background())
		done := make(chan struct{})
		go func() {
			defer close(done)
			for j := 0; j < 25; j++ {
				e.Submit("x")
			}
		}()
		go func() {
			time.Sleep(2 * time.Millisecond)
			e.Cancel()
		}()
		drainChFor(150*time.Millisecond, e.Events())
		<-done
		e.Wait()
	}
}
