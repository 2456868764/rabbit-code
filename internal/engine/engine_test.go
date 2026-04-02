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
		CompactAdvisor: func(_ query.LoopState, _ int) (bool, bool) {
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
