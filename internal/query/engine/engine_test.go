package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/2456868764/rabbit-code/internal/config"
	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/query"
	"github.com/2456868764/rabbit-code/internal/services/api"
	"github.com/2456868764/rabbit-code/internal/services/compact"
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

func TestEngine_SubmitWithOptions_commandLifecycleNotify_stub(t *testing.T) {
	var got [][2]string
	e := New(context.Background(), &Config{
		CommandLifecycleNotify: func(uuid, phase string) {
			got = append(got, [2]string{uuid, phase})
		},
	})
	e.SubmitWithOptions("hi", SubmitOptions{ConsumedCommandUUIDs: []string{" cmd-1 ", "", "cmd-2"}})
	ch := e.Events()
	for {
		select {
		case ev := <-ch:
			if ev.Kind == EventKindDone {
				e.Wait()
				if len(got) != 2 {
					t.Fatalf("want 2 notifications, got %+v", got)
				}
				if got[0][0] != "cmd-1" || got[0][1] != "completed" {
					t.Fatalf("first: %+v", got[0])
				}
				if got[1][0] != "cmd-2" || got[1][1] != "completed" {
					t.Fatalf("second: %+v", got[1])
				}
				return
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestEngine_SubmitWithOptions_commandLifecycleNotify_afterRunTurnLoop(t *testing.T) {
	var got [][2]string
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Turn: &query.SequenceTurnAssistant{Turns: []query.TurnResult{{Text: "ok"}}},
		},
		Model: "m", MaxTokens: 8,
		CommandLifecycleNotify: func(uuid, phase string) {
			got = append(got, [2]string{uuid, phase})
		},
	})
	e.SubmitWithOptions("hi", SubmitOptions{ConsumedCommandUUIDs: []string{"u1"}})
	ch := e.Events()
	for {
		select {
		case ev := <-ch:
			switch ev.Kind {
			case EventKindDone:
				e.Wait()
				if len(got) != 1 || got[0][0] != "u1" || got[0][1] != "completed" {
					t.Fatalf("got %+v", got)
				}
				return
			case EventKindError:
				t.Fatal(ev.Err)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestEngine_ProactiveAutoCompact_suggestWithoutAdvisor(t *testing.T) {
	t.Setenv(features.EnvContextWindowTokens, "50000")
	t.Setenv(features.EnvDisableCompact, "")
	t.Setenv(features.EnvDisableAutoCompact, "")
	t.Setenv(features.EnvAutoCompact, "")
	t.Setenv(features.EnvContextCollapse, "")
	t.Setenv(features.EnvSuppressProactiveAutoCompact, "")
	longReply := strings.Repeat("z", 150_000)
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return longReply, nil
			}),
		},
		Model: "m", MaxTokens: 1024,
		CompactExecutor: func(_ context.Context, phase compact.RunPhase, _ []byte) (string, []byte, error) {
			_ = phase
			return "auto-compact", nil, nil
		},
	})
	e.Submit("hi")
	ch := e.Events()
	deadline := time.After(3 * time.Second)
	var sawSuggest bool
	for {
		select {
		case ev := <-ch:
			switch ev.Kind {
			case EventKindCompactSuggest:
				if !ev.SuggestAutoCompact || ev.SuggestReactiveCompact {
					t.Fatalf("want auto-only suggest: %+v", ev)
				}
				sawSuggest = true
			case EventKindDone:
				if !sawSuggest {
					t.Fatal("expected compact suggest before done")
				}
				e.Wait()
				return
			case EventKindError:
				t.Fatal(ev.Err)
			}
		case <-deadline:
			t.Fatal("timeout")
		}
	}
}

func TestEngine_PostCompactCleanup_afterSuccessfulCompactExecutor(t *testing.T) {
	t.Setenv(features.EnvContextWindowTokens, "50000")
	t.Setenv(features.EnvReactiveCompactMinBytes, "999999999")
	t.Setenv(features.EnvReactiveCompactMinTokens, "999999999")
	t.Setenv(features.EnvDisableCompact, "")
	t.Setenv(features.EnvDisableAutoCompact, "")
	t.Setenv(features.EnvAutoCompact, "")
	t.Setenv(features.EnvContextCollapse, "")
	t.Setenv(features.EnvSuppressProactiveAutoCompact, "")
	var calls int32
	var sawMain bool
	longReply := strings.Repeat("z", 150_000)
	e := New(context.Background(), &Config{
		QuerySource: "repl_main_thread:test",
		AgentID:     "agent-postcompact",
		PostCompactCleanup: func(_ context.Context, querySource, agentID string, mainThread bool) {
			atomic.AddInt32(&calls, 1)
			if querySource != "repl_main_thread:test" || agentID != "agent-postcompact" {
				t.Errorf("unexpected args qs=%q agent=%q", querySource, agentID)
			}
			sawMain = mainThread
		},
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return longReply, nil
			}),
		},
		Model: "m", MaxTokens: 1024,
		CompactExecutor: func(_ context.Context, phase compact.RunPhase, _ []byte) (string, []byte, error) {
			_ = phase
			return "ok", nil, nil
		},
	})
	e.Submit("hi")
	drainEngineUntilDone(t, e.Events())
	e.Wait()
	if atomic.LoadInt32(&calls) != 1 {
		t.Fatalf("post-compact calls %d", calls)
	}
	if !sawMain {
		t.Fatal("expected main-thread post-compact")
	}
}

func TestEngine_SessionMemoryCompact_skipsLegacyAutoExecutor(t *testing.T) {
	t.Setenv(features.EnvContextWindowTokens, "50000")
	t.Setenv(features.EnvReactiveCompactMinBytes, "999999999")
	t.Setenv(features.EnvReactiveCompactMinTokens, "999999999")
	t.Setenv(features.EnvDisableCompact, "")
	t.Setenv(features.EnvDisableAutoCompact, "")
	t.Setenv(features.EnvAutoCompact, "")
	t.Setenv(features.EnvContextCollapse, "")
	t.Setenv(features.EnvSuppressProactiveAutoCompact, "")
	var legacyCalls int32
	longReply := strings.Repeat("z", 150_000)
	e := New(context.Background(), &Config{
		SessionMemoryCompact: func(ctx context.Context, agentID, model string, th int, transcript json.RawMessage) (json.RawMessage, bool, error) {
			_ = ctx
			_ = agentID
			_ = model
			_ = th
			_ = transcript
			return json.RawMessage(`[{"role":"user","content":[{"type":"text","text":"sm"}]}]`), true, nil
		},
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return longReply, nil
			}),
		},
		Model: "m", MaxTokens: 1024,
		CompactExecutor: func(_ context.Context, phase compact.RunPhase, _ []byte) (string, []byte, error) {
			_ = phase
			atomic.AddInt32(&legacyCalls, 1)
			return "legacy", nil, nil
		},
	})
	e.Submit("hi")
	var sawSM bool
	ch := e.Events()
	deadline := time.After(8 * time.Second)
	for {
		select {
		case ev := <-ch:
			if ev.Kind == EventKindCompactResult && ev.CompactSummary == "session_memory_compact" {
				sawSM = true
			}
			if ev.Kind == EventKindDone {
				e.Wait()
				if !sawSM {
					t.Fatal("expected session_memory_compact result")
				}
				if atomic.LoadInt32(&legacyCalls) != 0 {
					t.Fatalf("legacy executor should not run, calls=%d", legacyCalls)
				}
				return
			}
			if ev.Kind == EventKindError {
				t.Fatal(ev.Err)
			}
		case <-deadline:
			t.Fatal("timeout")
		}
	}
}

func TestEngine_AfterSessionMemoryCompactSuccess_onSMCompact(t *testing.T) {
	t.Setenv(features.EnvContextWindowTokens, "50000")
	t.Setenv(features.EnvReactiveCompactMinBytes, "999999999")
	t.Setenv(features.EnvReactiveCompactMinTokens, "999999999")
	t.Setenv(features.EnvDisableCompact, "")
	t.Setenv(features.EnvDisableAutoCompact, "")
	t.Setenv(features.EnvAutoCompact, "")
	t.Setenv(features.EnvContextCollapse, "")
	t.Setenv(features.EnvSuppressProactiveAutoCompact, "")
	var hookCalls int32
	longReply := strings.Repeat("z", 150_000)
	e := New(context.Background(), &Config{
		AfterSessionMemoryCompactSuccess: func(ctx context.Context, querySource, agentID string) {
			_ = ctx
			_ = querySource
			_ = agentID
			atomic.AddInt32(&hookCalls, 1)
		},
		SessionMemoryCompact: func(ctx context.Context, agentID, model string, th int, transcript json.RawMessage) (json.RawMessage, bool, error) {
			_ = ctx
			_ = agentID
			_ = model
			_ = th
			_ = transcript
			return json.RawMessage(`[{"role":"user","content":[{"type":"text","text":"sm"}]}]`), true, nil
		},
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return longReply, nil
			}),
		},
		Model: "m", MaxTokens: 1024,
	})
	e.Submit("hi")
	ch := e.Events()
	deadline := time.After(8 * time.Second)
	for {
		select {
		case ev := <-ch:
			if ev.Kind == EventKindDone {
				e.Wait()
				if atomic.LoadInt32(&hookCalls) != 1 {
					t.Fatalf("AfterSessionMemoryCompactSuccess calls=%d", hookCalls)
				}
				return
			}
			if ev.Kind == EventKindError {
				t.Fatal(ev.Err)
			}
		case <-deadline:
			t.Fatal("timeout")
		}
	}
}

func TestNew_defaultSessionMemoryCompactFromMemdir(t *testing.T) {
	ctx := context.Background()
	t.Setenv(features.EnvSessionMemoryFeature, "1")
	t.Setenv(features.EnvSessionMemoryCompactFeature, "1")
	t.Setenv(features.EnvDisableClaudeCodeSMCompact, "")
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "MEMORY.md"), []byte("# M\n\nbody"), 0o600); err != nil {
		t.Fatal(err)
	}
	e := New(ctx, &Config{
		MemdirMemoryDir: dir,
		InitialSettings: map[string]interface{}{"autoMemoryEnabled": true},
	})
	if e.sessionMemoryCompact == nil {
		t.Fatal("expected default SessionMemoryCompact from MEMORY.md")
	}
}

func TestNew_sessionMemoryCompactExplicitOverridesDefault(t *testing.T) {
	ctx := context.Background()
	t.Setenv(features.EnvSessionMemoryFeature, "1")
	t.Setenv(features.EnvSessionMemoryCompactFeature, "1")
	var called bool
	explicit := func(context.Context, string, string, int, json.RawMessage) (json.RawMessage, bool, error) {
		called = true
		return nil, false, nil
	}
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "MEMORY.md"), []byte("x"), 0o600)
	e := New(ctx, &Config{
		MemdirMemoryDir:      dir,
		InitialSettings:      map[string]interface{}{"autoMemoryEnabled": true},
		SessionMemoryCompact: explicit,
	})
	_, _, _ = e.sessionMemoryCompact(context.Background(), "", "", 0, json.RawMessage(`[]`))
	if !called {
		t.Fatal("explicit SessionMemoryCompact should be kept")
	}
}

func TestRunCompactSuggestAfterSuccessfulTurn_respectsDisableCompact(t *testing.T) {
	t.Setenv(features.EnvDisableCompact, "1")
	t.Setenv(features.EnvSessionMemoryFeature, "")
	t.Setenv(features.EnvSessionMemoryCompactFeature, "")
	var advisorCalls int
	e := New(context.Background(), &Config{
		CompactAdvisor: func(query.LoopState, []byte) (bool, bool) {
			advisorCalls++
			return true, true
		},
	})
	st := &query.LoopState{}
	_ = e.runCompactSuggestAfterSuccessfulTurn(st, json.RawMessage(`[]`))
	if advisorCalls != 0 {
		t.Fatalf("advisor should not run when DISABLE_COMPACT: calls=%d", advisorCalls)
	}
}

func drainEngineUntilDone(t *testing.T, ch <-chan EngineEvent) {
	t.Helper()
	deadline := time.After(8 * time.Second)
	for {
		select {
		case ev := <-ch:
			if ev.Kind == EventKindDone {
				return
			}
			if ev.Kind == EventKindError {
				t.Fatal(ev.Err)
			}
		case <-deadline:
			t.Fatal("timeout waiting for EventKindDone")
		}
	}
}

func TestEngine_AutoCompactCircuit_tripsAfterExecutorFailures(t *testing.T) {
	t.Setenv(features.EnvContextWindowTokens, "50000")
	t.Setenv(features.EnvReactiveCompactMinBytes, "999999999")
	t.Setenv(features.EnvReactiveCompactMinTokens, "999999999")
	t.Setenv(features.EnvDisableCompact, "")
	t.Setenv(features.EnvDisableAutoCompact, "")
	t.Setenv(features.EnvAutoCompact, "")
	t.Setenv(features.EnvContextCollapse, "")
	t.Setenv(features.EnvSuppressProactiveAutoCompact, "")
	longReply := strings.Repeat("z", 150_000)
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return longReply, nil
			}),
		},
		Model: "m", MaxTokens: 1024,
		CompactExecutor: func(_ context.Context, phase compact.RunPhase, _ []byte) (string, []byte, error) {
			_ = phase
			return "", nil, errors.New("stub compact failure")
		},
	})
	for range 3 {
		e.Submit("hi")
		drainEngineUntilDone(t, e.Events())
		e.Wait()
	}
	e.Submit("hi")
	ch := e.Events()
	deadline := time.After(8 * time.Second)
	for {
		select {
		case ev := <-ch:
			if ev.Kind == EventKindCompactSuggest && ev.SuggestAutoCompact {
				t.Fatalf("circuit should block proactive auto after 3 failures: %+v", ev)
			}
			if ev.Kind == EventKindDone {
				e.Wait()
				return
			}
			if ev.Kind == EventKindError {
				t.Fatal(ev.Err)
			}
		case <-deadline:
			t.Fatal("timeout")
		}
	}
}

func TestEngine_ProactiveAutoCompact_suppressedForQuerySourceSessionMemory(t *testing.T) {
	t.Setenv(features.EnvContextWindowTokens, "50000")
	t.Setenv(features.EnvReactiveCompactMinBytes, "999999999")
	t.Setenv(features.EnvReactiveCompactMinTokens, "999999999")
	t.Setenv(features.EnvDisableCompact, "")
	t.Setenv(features.EnvDisableAutoCompact, "")
	t.Setenv(features.EnvAutoCompact, "")
	t.Setenv(features.EnvContextCollapse, "")
	t.Setenv(features.EnvSuppressProactiveAutoCompact, "")
	longReply := strings.Repeat("z", 150_000)
	e := New(context.Background(), &Config{
		QuerySource: query.QuerySourceSessionMemory,
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return longReply, nil
			}),
		},
		Model: "m", MaxTokens: 1024,
		CompactExecutor: func(_ context.Context, phase compact.RunPhase, _ []byte) (string, []byte, error) {
			_ = phase
			return "should-not-run", nil, nil
		},
	})
	e.Submit("hi")
	ch := e.Events()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case ev := <-ch:
			switch ev.Kind {
			case EventKindCompactSuggest:
				if ev.SuggestAutoCompact {
					t.Fatalf("auto compact should not suggest for session_memory fork: %+v", ev)
				}
			case EventKindCompactResult:
				t.Fatalf("unexpected compact result (reactive should be off): %+v", ev)
			case EventKindDone:
				e.Wait()
				return
			case EventKindError:
				t.Fatal(ev.Err)
			}
		case <-deadline:
			t.Fatal("timeout")
		}
	}
}

func TestEngine_CompactE2E_longTranscriptTriggersExecutor(t *testing.T) {
	longReply := strings.Repeat("xy ", 400)
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return longReply, nil
			}),
		},
		Model: "m", MaxTokens: 64,
		CompactAdvisor: func(_ query.LoopState, transcriptJSON []byte) (bool, bool) {
			return len(transcriptJSON) > 1200, false
		},
		CompactExecutor: func(_ context.Context, phase compact.RunPhase, _ []byte) (string, []byte, error) {
			_ = phase
			return "e2e-compact-summary", nil, nil
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
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
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

func TestEngine_TokenBudget_combinedTextAndInjectRawTokens(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "eight.txt")
	if err := os.WriteFile(p, []byte("12345678"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(features.EnvTokenBudget, "true")
	t.Setenv(features.EnvTokenBudgetMaxInputBytes, "999999")
	t.Setenv(features.EnvTokenBudgetMaxAttachmentBytes, "999999")
	t.Setenv(features.EnvTokenBudgetMaxInputTokens, "2")
	e := New(context.Background(), &Config{
		MemdirPaths: []string{p},
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
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
				goto combDone
			}
			if ev.Kind == EventKindAssistantText {
				t.Fatal("assistant should not run")
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
	}
combDone:
	e.Wait()
	if !errors.Is(saw, ErrTokenBudgetExceeded) {
		t.Fatalf("want combined text+inject over cap, got %v", saw)
	}
}

func TestEngine_TokenBudget_emitsSubmitSnapshot(t *testing.T) {
	t.Setenv(features.EnvTokenBudget, "true")
	t.Setenv(features.EnvTokenBudgetMaxInputBytes, "999999")
	t.Setenv(features.EnvTokenBudgetMaxInputTokens, "999999")
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "ok", nil
			}),
		},
	})
	e.Submit("hi")
	var sawSnap bool
	for {
		select {
		case ev := <-e.Events():
			if ev.Kind == EventKindSubmitTokenBudgetSnapshot {
				sawSnap = true
				if ev.PhaseAuxInt <= 0 {
					t.Fatalf("expected positive PhaseAuxInt, got %d", ev.PhaseAuxInt)
				}
				if ev.PhaseDetail != "bytes4" && ev.PhaseDetail != "structured" {
					t.Fatalf("mode %q", ev.PhaseDetail)
				}
			}
			if ev.Kind == EventKindDone {
				if !sawSnap {
					t.Fatal("missing SubmitTokenBudgetSnapshot")
				}
				e.Wait()
				return
			}
		case <-time.After(3 * time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestEngine_TokenBudget_blocksByTokenEstimate(t *testing.T) {
	t.Setenv(features.EnvTokenBudget, "true")
	t.Setenv(features.EnvTokenBudgetMaxInputBytes, "999999")
	t.Setenv(features.EnvTokenBudgetMaxInputTokens, "1")
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
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
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
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
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(ctx context.Context, model string, maxTokens int, messagesJSON []byte) (string, error) {
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
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
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
	turns := &query.SequenceTurnAssistant{Turns: []query.TurnResult{
		{Text: "a", ToolUses: []query.ToolUseCall{{ID: "1", Name: "bash", Input: json.RawMessage(`{}`)}}},
		{Text: "b"},
	}}
	e := New(context.Background(), &Config{
		Deps:  query.Deps{Tools: tr, Turn: turns},
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
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(_ context.Context, _ string, _ int, messagesJSON []byte) (string, error) {
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

func TestEngine_MemdirMemoryDir_heuristicAndSurfacedExcludesSecondSubmit(t *testing.T) {
	memDir := t.TempDir()
	md := filepath.Join(memDir, "note.md")
	if err := os.WriteFile(md, []byte("everything about bananas"), 0o644); err != nil {
		t.Fatal(err)
	}
	var memdirCount int
	var lastUser string
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(_ context.Context, _ string, _ int, messagesJSON []byte) (string, error) {
				lastUser = string(messagesJSON)
				return "ok", nil
			}),
		},
		MemdirMemoryDir:             memDir,
		MemdirRelevanceModeOverride: "heuristic",
	})
	drain := func() {
		for {
			select {
			case ev := <-e.Events():
				if ev.Kind == EventKindMemdirInject {
					memdirCount++
				}
				if ev.Kind == EventKindDone {
					return
				}
			case <-time.After(3 * time.Second):
				t.Fatal("timeout")
			}
		}
	}
	e.Submit("banana bread")
	drain()
	if memdirCount != 1 {
		t.Fatalf("first submit memdir events %d", memdirCount)
	}
	if !strings.Contains(lastUser, "bananas") {
		t.Fatalf("missing fragment: %q", lastUser)
	}
	e.Submit("banana bread again")
	drain()
	if memdirCount != 1 {
		t.Fatalf("second submit should not inject again; total memdir events %d", memdirCount)
	}
	e.Wait()
}

func TestResolveEngineMemdirMemoryDir_env(t *testing.T) {
	memDir := t.TempDir()
	t.Setenv(features.EnvMemdirMemoryDir, memDir)
	if got := resolveEngineMemdirMemoryDir(&Config{}); got != memDir {
		t.Fatalf("resolveEngineMemdirMemoryDir: got %q want %q", got, memDir)
	}
}

func TestEngine_MemdirMemoryDir_fromEnv(t *testing.T) {
	memDir := t.TempDir()
	t.Setenv(features.EnvMemdirMemoryDir, memDir)
	md := filepath.Join(memDir, "note.md")
	if err := os.WriteFile(md, []byte("everything about bananas"), 0o644); err != nil {
		t.Fatal(err)
	}
	var lastUser string
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(_ context.Context, _ string, _ int, messagesJSON []byte) (string, error) {
				lastUser = string(messagesJSON)
				return "ok", nil
			}),
		},
		MemdirRelevanceModeOverride: "heuristic",
	})
	e.Submit("banana bread")
	for {
		select {
		case ev := <-e.Events():
			if ev.Kind == EventKindDone {
				goto doneEnvMem
			}
		case <-time.After(3 * time.Second):
			t.Fatal("timeout")
		}
	}
doneEnvMem:
	e.Wait()
	if !strings.Contains(lastUser, "bananas") {
		t.Fatalf("missing fragment: %q", lastUser)
	}
}

func TestEngine_MemdirMemoryDir_autoResolve_memoryPathOverride(t *testing.T) {
	memDir := t.TempDir()
	t.Setenv("RABBIT_CODE_MEMORY_PATH_OVERRIDE", memDir)
	t.Setenv(features.EnvAutoMemdir, "1")
	proj := t.TempDir()
	md := filepath.Join(memDir, "note.md")
	if err := os.WriteFile(md, []byte("everything about plums"), 0o644); err != nil {
		t.Fatal(err)
	}
	var lastUser string
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(_ context.Context, _ string, _ int, messagesJSON []byte) (string, error) {
				lastUser = string(messagesJSON)
				return "ok", nil
			}),
		},
		MemdirProjectRoot:           proj,
		MemdirRelevanceModeOverride: "heuristic",
	})
	e.Submit("plum jam")
	for {
		select {
		case ev := <-e.Events():
			if ev.Kind == EventKindDone {
				goto doneAuto
			}
		case <-time.After(3 * time.Second):
			t.Fatal("timeout")
		}
	}
doneAuto:
	e.Wait()
	if !strings.Contains(lastUser, "plums") {
		t.Fatalf("missing fragment: %q", lastUser)
	}
}

func TestResolveEngineMemdirMemoryDir_trustedWithoutAutoMemdirEnv(t *testing.T) {
	t.Setenv(features.EnvAutoMemdir, "")
	memDir := t.TempDir()
	got := resolveEngineMemdirMemoryDir(&Config{MemdirTrustedAutoMemoryDirectory: memDir})
	want, err := filepath.Abs(memDir)
	if err != nil {
		t.Fatal(err)
	}
	gotAbs, err := filepath.Abs(got)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Clean(gotAbs) != filepath.Clean(want) {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestEngine_MemdirMemoryDir_trustedConfigOnly(t *testing.T) {
	memDir := t.TempDir()
	t.Setenv(features.EnvAutoMemdir, "")
	md := filepath.Join(memDir, "note.md")
	if err := os.WriteFile(md, []byte("everything about bananas"), 0o644); err != nil {
		t.Fatal(err)
	}
	var lastUser string
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(_ context.Context, _ string, _ int, messagesJSON []byte) (string, error) {
				lastUser = string(messagesJSON)
				return "ok", nil
			}),
		},
		MemdirTrustedAutoMemoryDirectory: memDir,
		MemdirRelevanceModeOverride:      "heuristic",
	})
	e.Submit("banana bread")
	for {
		select {
		case ev := <-e.Events():
			if ev.Kind == EventKindDone {
				goto doneTrustedOnly
			}
		case <-time.After(3 * time.Second):
			t.Fatal("timeout")
		}
	}
doneTrustedOnly:
	e.Wait()
	if !strings.Contains(lastUser, "bananas") {
		t.Fatalf("missing fragment: %q", lastUser)
	}
}

func TestEngine_MemdirMemoryDir_fromLoadTrustedAutoMemoryDirectory(t *testing.T) {
	root := t.TempDir()
	global := t.TempDir()
	memDir := t.TempDir()
	user := filepath.Join(global, config.UserConfigFileName)
	if err := os.WriteFile(user, []byte(fmt.Sprintf(`{"autoMemoryDirectory":%q}`, memDir)), 0o600); err != nil {
		t.Fatal(err)
	}
	trusted, err := config.LoadTrustedAutoMemoryDirectory(config.Paths{
		GlobalConfigDir: global,
		ProjectRoot:     root,
	})
	if err != nil {
		t.Fatal(err)
	}
	if trusted != memDir {
		t.Fatalf("trusted %q", trusted)
	}
	if err := os.WriteFile(filepath.Join(root, ".rabbit-code.json"), []byte(fmt.Sprintf(`{"autoMemoryDirectory":%q}`, filepath.Join(root, "evil"))), 0o600); err != nil {
		t.Fatal(err)
	}
	trusted2, err := config.LoadTrustedAutoMemoryDirectory(config.Paths{
		GlobalConfigDir: global,
		ProjectRoot:     root,
	})
	if err != nil || trusted2 != memDir {
		t.Fatalf("after project file trusted2=%q err=%v", trusted2, err)
	}

	md := filepath.Join(memDir, "note.md")
	if err := os.WriteFile(md, []byte("everything about bananas"), 0o644); err != nil {
		t.Fatal(err)
	}
	var lastUser string
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(_ context.Context, _ string, _ int, messagesJSON []byte) (string, error) {
				lastUser = string(messagesJSON)
				return "ok", nil
			}),
		},
		MemdirTrustedAutoMemoryDirectory: trusted,
		MemdirRelevanceModeOverride:      "heuristic",
	})
	e.Submit("banana bread")
	for {
		select {
		case ev := <-e.Events():
			if ev.Kind == EventKindDone {
				goto doneLoadTrusted
			}
		case <-time.After(3 * time.Second):
			t.Fatal("timeout")
		}
	}
doneLoadTrusted:
	e.Wait()
	if !strings.Contains(lastUser, "bananas") {
		t.Fatalf("missing fragment: %q", lastUser)
	}
}

func TestEngine_Memdir_initialSettings_autoMemoryDisabled(t *testing.T) {
	memDir := t.TempDir()
	t.Setenv(features.EnvAutoMemdir, "")
	md := filepath.Join(memDir, "note.md")
	if err := os.WriteFile(md, []byte("everything about bananas"), 0o644); err != nil {
		t.Fatal(err)
	}
	var lastUser string
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(_ context.Context, _ string, _ int, messagesJSON []byte) (string, error) {
				lastUser = string(messagesJSON)
				return "ok", nil
			}),
		},
		MemdirTrustedAutoMemoryDirectory: memDir,
		InitialSettings:                  map[string]interface{}{"autoMemoryEnabled": false},
		MemdirRelevanceModeOverride:      "heuristic",
	})
	e.Submit("banana bread")
	for {
		select {
		case ev := <-e.Events():
			if ev.Kind == EventKindDone {
				goto doneOff
			}
		case <-time.After(3 * time.Second):
			t.Fatal("timeout")
		}
	}
doneOff:
	e.Wait()
	if strings.Contains(lastUser, "bananas") {
		t.Fatalf("memdir should be off: %q", lastUser)
	}
}

func TestEngine_CompactSuggest_afterLoop(t *testing.T) {
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
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
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
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
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
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
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
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
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
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
	seq := &query.SequenceTurnAssistant{Turns: []query.TurnResult{
		{Text: "t1", ToolUses: []query.ToolUseCall{{ID: "x", Name: "bash", Input: json.RawMessage(`{}`)}}},
		{Text: "t2"},
	}}
	e := New(context.Background(), &Config{
		Deps: query.Deps{
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
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
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
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
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
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
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
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
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
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
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
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
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
	turns := &query.SequenceTurnAssistant{Turns: []query.TurnResult{
		{Text: "a"},
		{Text: "b"},
	}}
	var nCont int
	e := New(context.Background(), &Config{
		Deps:  query.Deps{Turn: turns},
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
	turns := &query.SequenceTurnAssistant{Turns: []query.TurnResult{
		{Text: "t1"},
		{Text: "t2"},
	}}
	var nTok int
	var captured query.LoopState
	e := New(context.Background(), &Config{
		Deps:  query.Deps{Turn: turns},
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
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
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
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(_ context.Context, _ string, _ int, messagesJSON []byte) (string, error) {
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

func TestEngine_RecoverStrategy_compactNextTranscriptSeedsRetry(t *testing.T) {
	apiErr := &anthropic.APIError{Kind: anthropic.KindPromptTooLong, Status: 400, Msg: "ptl"}
	nextMsgs, err := query.InitialUserMessagesJSON("COMPACT_RETRY_MARKER")
	if err != nil {
		t.Fatal(err)
	}
	var n int
	var secondBody string
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(_ context.Context, _ string, _ int, messagesJSON []byte) (string, error) {
				n++
				if n == 1 {
					return "", fmt.Errorf("w: %w", apiErr)
				}
				secondBody = string(messagesJSON)
				return "ok", nil
			}),
		},
		SuggestCompactOnRecoverableError: true,
		RecoverStrategy:                  func(context.Context, query.LoopState, error) bool { return true },
		CompactExecutor: func(_ context.Context, _ compact.RunPhase, _ []byte) (string, []byte, error) {
			return "compact-ok", nextMsgs, nil
		},
	})
	e.Submit("first-user-should-not-retry")
	drainUntilTerminal(t, e.Events())
	e.Wait()
	if n != 2 {
		t.Fatalf("want 2 assistant calls, got %d", n)
	}
	if !strings.Contains(secondBody, "COMPACT_RETRY_MARKER") || strings.Contains(secondBody, "first-user-should-not-retry") {
		t.Fatalf("retry should use executor next transcript; got %q", secondBody)
	}
}

func TestEngine_drainThenCompact_compactNextWinsOnRetrySeed(t *testing.T) {
	t.Setenv(features.EnvContextCollapse, "true")
	apiErr := &anthropic.APIError{Kind: anthropic.KindPromptTooLong, Status: 400, Msg: "ptl"}
	drained, err := query.InitialUserMessagesJSON("DRAIN_ONLY")
	if err != nil {
		t.Fatal(err)
	}
	compactOut, err := query.InitialUserMessagesJSON("COMPACT_WINS_SEED")
	if err != nil {
		t.Fatal(err)
	}
	var n int
	var secondBody string
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(_ context.Context, _ string, _ int, messagesJSON []byte) (string, error) {
				n++
				if n == 1 {
					return "", fmt.Errorf("w: %w", apiErr)
				}
				secondBody = string(messagesJSON)
				return "ok", nil
			}),
		},
		SuggestCompactOnRecoverableError: true,
		RecoverStrategy:                  func(context.Context, query.LoopState, error) bool { return true },
		ContextCollapseDrain: func(_ context.Context, _ *query.LoopState, _ json.RawMessage) (json.RawMessage, int, bool) {
			return drained, 1, true
		},
		CompactExecutor: func(_ context.Context, _ compact.RunPhase, _ []byte) (string, []byte, error) {
			return "s", compactOut, nil
		},
	})
	e.Submit("original")
	drainUntilTerminal(t, e.Events())
	e.Wait()
	if !strings.Contains(secondBody, "COMPACT_WINS_SEED") || strings.Contains(secondBody, "DRAIN_ONLY") {
		t.Fatalf("compact next transcript should win over drain-only seed; got %q", secondBody)
	}
}

func TestEngine_ConfigAgentIDNonInteractive_inLoopState(t *testing.T) {
	var captured query.LoopState
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "x", nil
			}),
		},
		Model:          "custom-model",
		AgentID:        "engine-agent",
		NonInteractive: true,
		SessionID:      " sid ",
		Debug:          true,
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
	if captured.ToolUseContext.SessionID != "sid" || !captured.ToolUseContext.Debug {
		t.Fatalf("%+v", captured.ToolUseContext)
	}
	if len(captured.MessagesJSON) == 0 || !strings.Contains(string(captured.MessagesJSON), "hi") {
		t.Fatalf("MessagesJSON %s", captured.MessagesJSON)
	}
}

func TestEngine_StopHooksAfterSuccessfulTurn_preventContinuation(t *testing.T) {
	t.Setenv(features.EnvTokenBudget, "true")
	t.Setenv(features.EnvTokenBudgetMaxInputBytes, "999999")
	var tokenCalls int
	var captured query.LoopState
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "ok", nil
			}),
		},
		StopHooksAfterSuccessfulTurn: []StopHookAfterTurnFunc{
			func(context.Context, query.LoopState, json.RawMessage) StopHookAfterTurnResult {
				return StopHookAfterTurnResult{PreventContinuation: true, StopReason: "user-stop"}
			},
		},
		TokenBudgetContinueAfterTurn: func(context.Context, query.LoopState, json.RawMessage) bool {
			tokenCalls++
			return true
		},
		StopHooks: []StopHookFunc{
			func(_ context.Context, st query.LoopState, _ error) { captured = st },
		},
	})
	e.Submit("x")
	ch := e.Events()
	deadline := time.After(2 * time.Second)
	var sawDone bool
	for !sawDone {
		select {
		case ev := <-ch:
			if ev.Kind == EventKindDone {
				sawDone = true
				if ev.PhaseDetail != "user-stop" {
					t.Fatalf("PhaseDetail %q", ev.PhaseDetail)
				}
			}
			if ev.Kind == EventKindError {
				t.Fatal(ev.Err)
			}
		case <-deadline:
			t.Fatal("timeout")
		}
	}
	e.Wait()
	if tokenCalls != 0 {
		t.Fatalf("token hook should not run after prevent: %d", tokenCalls)
	}
	if captured.LoopContinue.Reason != query.ContinueReasonStopHookPrevented {
		t.Fatalf("%+v", captured.LoopContinue)
	}
}

func TestEngine_StopHooksAfterSuccessfulTurn_blockingContinue(t *testing.T) {
	turns := &query.SequenceTurnAssistant{Turns: []query.TurnResult{
		{Text: "a"},
		{Text: "b"},
	}}
	var afterCalls int
	e := New(context.Background(), &Config{
		Deps:  query.Deps{Turn: turns},
		Model: "m", MaxTokens: 8,
		StopHooksAfterSuccessfulTurn: []StopHookAfterTurnFunc{
			func(context.Context, query.LoopState, json.RawMessage) StopHookAfterTurnResult {
				afterCalls++
				if afterCalls == 1 {
					return StopHookAfterTurnResult{BlockingContinue: true}
				}
				return StopHookAfterTurnResult{}
			},
		},
	})
	e.Submit("hi")
	drainUntilTerminal(t, e.Events())
	e.Wait()
	if afterCalls != 2 {
		t.Fatalf("after-turn hooks %d", afterCalls)
	}
}

func TestEngine_BashStubToolRunner(t *testing.T) {
	turns := &query.SequenceTurnAssistant{Turns: []query.TurnResult{
		{Text: "run", ToolUses: []query.ToolUseCall{{ID: "t1", Name: "bash", Input: json.RawMessage(`{"cmd":"ls"}`)}}},
		{Text: "done"},
	}}
	e := New(context.Background(), &Config{
		Deps:  query.Deps{Tools: query.BashStubToolRunner{}, Turn: turns},
		Model: "m", MaxTokens: 8,
	})
	e.Submit("hi")
	drainUntilTerminal(t, e.Events())
	e.Wait()
}

func TestEngine_ToolCallFailed_emitsOrphanFromError(t *testing.T) {
	turns := &query.SequenceTurnAssistant{Turns: []query.TurnResult{
		{Text: "x", ToolUses: []query.ToolUseCall{{ID: "orph1", Name: "bash", Input: json.RawMessage(`{}`)}}},
	}}
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Turn: turns,
			Tools: toolRunnerFunc(func(_ context.Context, _ string, _ []byte) ([]byte, error) {
				return nil, &query.OrphanPermissionError{ToolUseID: "orph1"}
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
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "y", nil
			}),
		},
		CompactAdvisor: func(_ query.LoopState, _ []byte) (bool, bool) { return true, false },
		CompactExecutor: func(_ context.Context, phase compact.RunPhase, transcript []byte) (string, []byte, error) {
			_ = phase
			if len(transcript) == 0 {
				return "", nil, errors.New("empty transcript")
			}
			return "summary-ok", nil, nil
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
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
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

func TestEngine_headlessEnv_breakCacheTemplatesMicrocompactEvents(t *testing.T) {
	t.Setenv(features.EnvBreakCacheCommand, "true")
	t.Setenv(features.EnvTemplates, "true")
	t.Setenv(features.EnvTemplateNames, "a,b")
	t.Setenv(features.EnvCachedMicrocompact, "true")
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
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
				if ev.PhaseDetail != anthropic.BetaCachedMicrocompactBody {
					t.Fatalf("micro PhaseDetail: got %q want %q", ev.PhaseDetail, anthropic.BetaCachedMicrocompactBody)
				}
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

func TestEngine_AfterToolResultsHook(t *testing.T) {
	var calls int
	turns := &query.SequenceTurnAssistant{Turns: []query.TurnResult{
		{Text: "t", ToolUses: []query.ToolUseCall{{ID: "1", Name: "bash", Input: json.RawMessage(`{}`)}}},
		{Text: "done"},
	}}
	e := New(context.Background(), &Config{
		Deps:  query.Deps{Turn: turns, Tools: query.BashStubToolRunner{}},
		Model: "m", MaxTokens: 8,
		AfterToolResultsHook: func(ctx context.Context, st *query.LoopState, raw json.RawMessage) error {
			_ = ctx
			_ = st
			calls++
			return nil
		},
	})
	e.Submit("hi")
	ch := e.Events()
	deadline := time.After(3 * time.Second)
	for {
		select {
		case ev := <-ch:
			switch ev.Kind {
			case EventKindDone:
				e.Wait()
				if calls != 1 {
					t.Fatalf("want 1 hook call, got %d", calls)
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

func TestEngine_ProcessUserInputHook_replace(t *testing.T) {
	var captured string
	e := New(context.Background(), &Config{
		ProcessUserInputHook: func(_ context.Context, s string) (string, bool, error) {
			if s != "ORIG" {
				t.Fatalf("hook got %q", s)
			}
			return "REPLACED", true, nil
		},
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(_ context.Context, _ string, _ int, messagesJSON []byte) (string, error) {
				captured = string(messagesJSON)
				return "x", nil
			}),
		},
	})
	e.Submit("ORIG")
	for {
		select {
		case ev := <-e.Events():
			if ev.Kind == EventKindDone {
				e.Wait()
				if captured == "" || !strings.Contains(captured, "REPLACED") {
					t.Fatalf("messages should contain replaced text: %q", captured)
				}
				return
			}
			if ev.Kind == EventKindError {
				t.Fatal(ev.Err)
			}
		case <-time.After(2 * time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestEngine_ExtraTemplateNames_mergedAppendixAndEvent(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte("AAA"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "x.md"), []byte("XXX"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(features.EnvTemplates, "true")
	t.Setenv(features.EnvTemplateNames, "a")
	var captured string
	var tplDetail string
	e := New(context.Background(), &Config{
		TemplateDir: dir,
		ExtraTemplateNames: func(string) []string {
			return []string{"x"}
		},
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(_ context.Context, _ string, _ int, messagesJSON []byte) (string, error) {
				captured = string(messagesJSON)
				return "ok", nil
			}),
		},
	})
	e.Submit("u")
	for {
		select {
		case ev := <-e.Events():
			switch ev.Kind {
			case EventKindTemplatesActive:
				tplDetail = ev.PhaseDetail
			case EventKindDone:
				e.Wait()
				if tplDetail != "a,x" {
					t.Fatalf("TemplatesActive want a,x got %q", tplDetail)
				}
				if !strings.Contains(captured, "## Template a") || !strings.Contains(captured, "AAA") ||
					!strings.Contains(captured, "## Template x") || !strings.Contains(captured, "XXX") {
					t.Fatalf("messages %q", captured)
				}
				return
			case EventKindError:
				t.Fatal(ev.Err)
			}
		case <-time.After(2 * time.Second):
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
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(_ context.Context, _ string, _ int, messagesJSON []byte) (string, error) {
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

func TestEngine_UserSubmit_carriesHeadlessModeTags(t *testing.T) {
	t.Setenv(features.EnvUltrathink, "true")
	t.Setenv(features.EnvUltraplan, "true")
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
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

func TestEngine_ultrathinkInjectsIntoMessagesJSON(t *testing.T) {
	t.Setenv(features.EnvUltrathink, "true")
	var captured string
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(_ context.Context, _ string, _ int, messagesJSON []byte) (string, error) {
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

func TestEngine_reactiveCompactFromEnv_minTokens(t *testing.T) {
	t.Setenv(features.EnvReactiveCompact, "true")
	t.Setenv(features.EnvReactiveCompactMinBytes, "999999")
	t.Setenv(features.EnvReactiveCompactMinTokens, "1")
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
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

func TestEngine_reactiveCompactFromEnv(t *testing.T) {
	t.Setenv(features.EnvReactiveCompact, "true")
	t.Setenv(features.EnvReactiveCompactMinBytes, "1")
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
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

func TestEngine_historySnipBetweenRounds(t *testing.T) {
	t.Setenv(features.EnvHistorySnip, "true")
	t.Setenv(features.EnvHistorySnipMaxBytes, "280")
	t.Setenv(features.EnvHistorySnipMaxRounds, "2")
	long := strings.Repeat("z", 350)
	seq := &query.SequenceTurnAssistant{Turns: []query.TurnResult{
		{Text: long, ToolUses: []query.ToolUseCall{{ID: "1", Name: "bash", Input: json.RawMessage(`{}`)}}},
		{Text: "end"},
	}}
	e := New(context.Background(), &Config{
		Deps:  query.Deps{Turn: seq, Tools: query.BashStubToolRunner{}},
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
				if ev.SnipID == "" {
					t.Fatal("expected SnipID on history snip event")
				}
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

func TestEngine_SnipRemovalLogForPersistence_includesRestored(t *testing.T) {
	prev := []query.SnipRemovalEntry{{ID: "r1", Kind: query.SnipRemovalKindSnipCompact, RemovedMessageCount: 2}}
	e := New(context.Background(), &Config{
		RestoredSnipRemovalLog: prev,
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "ok", nil
			}),
		},
	})
	e.Submit("x")
	drainUntilTerminal(t, e.Events())
	e.Wait()
	got := e.SnipRemovalLogForPersistence()
	if len(got) != 1 || got[0].ID != "r1" {
		t.Fatalf("got %+v", got)
	}
}

func TestEngine_LastAssistantAtForPersistence_restoredAndUpdated(t *testing.T) {
	restored := time.Date(2025, 11, 10, 15, 30, 0, 0, time.UTC)
	e := New(context.Background(), &Config{
		RestoredSessionLastAssistantAt: restored,
		Deps: query.Deps{
			Turn: query.StreamAsTurnAssistant(query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "hi", nil
			})),
		},
	})
	if !e.LastAssistantAtForPersistence().Equal(restored) {
		t.Fatalf("before submit: got %v want %v", e.LastAssistantAtForPersistence(), restored)
	}
	e.Submit("hello")
	drainUntilTerminal(t, e.Events())
	e.Wait()
	after := e.LastAssistantAtForPersistence()
	if after.IsZero() {
		t.Fatal("expected non-zero after assistant turn")
	}
	if !after.After(restored) {
		t.Fatalf("expected updated wall clock after submit, got %v (restored %v)", after, restored)
	}
}

func TestEngine_SnipCompactBetweenRounds(t *testing.T) {
	t.Setenv(features.EnvSnipCompact, "true")
	t.Setenv(features.EnvSnipCompactMaxBytes, "280")
	t.Setenv(features.EnvSnipCompactMaxRounds, "2")
	long := strings.Repeat("s", 350)
	seq := &query.SequenceTurnAssistant{Turns: []query.TurnResult{
		{Text: long, ToolUses: []query.ToolUseCall{{ID: "1", Name: "bash", Input: json.RawMessage(`{}`)}}},
		{Text: "end"},
	}}
	e := New(context.Background(), &Config{
		Deps:  query.Deps{Turn: seq, Tools: query.BashStubToolRunner{}},
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

func (cacheBreakFireTurn) AssistantTurn(ctx context.Context, model string, maxTokens int, msgs []byte) (query.TurnResult, error) {
	_ = model
	_ = maxTokens
	_ = msgs
	if cb, ok := anthropic.OnPromptCacheBreakFromContext(ctx); ok && cb != nil {
		cb()
	}
	return query.TurnResult{Text: "ok"}, nil
}

type failOncePromptCacheBreakTurn struct {
	n int
}

func (f *failOncePromptCacheBreakTurn) AssistantTurn(ctx context.Context, model string, maxTokens int, msgs []byte) (query.TurnResult, error) {
	f.n++
	if f.n == 1 {
		return query.TurnResult{}, anthropic.ErrPromptCacheBreakDetected
	}
	return query.TurnResult{Text: "ok"}, nil
}

func TestEngine_promptCacheBreakAutoCompact_recovery(t *testing.T) {
	t.Setenv(features.EnvPromptCacheBreak, "1")
	t.Setenv(features.EnvPromptCacheBreakTrimResend, "0")
	t.Setenv(features.EnvPromptCacheBreakAutoCompact, "1")
	var sawCompactRetry bool
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Turn: &failOncePromptCacheBreakTurn{},
		},
		CompactExecutor: func(ctx context.Context, phase compact.RunPhase, transcriptJSON []byte) (string, []byte, error) {
			_ = ctx
			_ = phase
			_ = transcriptJSON
			return "s", []byte(`[{"role":"user","content":[{"type":"text","text":"after-compact"}]}]`), nil
		},
	})
	e.Submit("hi")
	for {
		select {
		case ev := <-e.Events():
			if ev.Kind == EventKindPromptCacheBreakRecovery && ev.PhaseDetail == "compact_retry" {
				sawCompactRetry = true
			}
			if ev.Kind == EventKindDone {
				if !sawCompactRetry {
					t.Fatal("expected compact_retry recovery event")
				}
				e.Wait()
				return
			}
			if ev.Kind == EventKindError {
				t.Fatalf("error: %v", ev.Err)
			}
		case <-time.After(3 * time.Second):
			t.Fatal("timeout")
		}
	}
}

type failTwicePromptCacheBreakThenOK struct {
	n int
	t *testing.T
}

func (f *failTwicePromptCacheBreakThenOK) AssistantTurn(ctx context.Context, model string, maxTokens int, msgs []byte) (query.TurnResult, error) {
	f.n++
	if f.n <= 2 {
		return query.TurnResult{}, anthropic.ErrPromptCacheBreakDetected
	}
	if !bytes.Contains(msgs, []byte("seed2")) {
		f.t.Fatalf("AssistantTurn call %d: want transcript from second compact, got %s", f.n, msgs)
	}
	return query.TurnResult{Text: "h1-two-compacts"}, nil
}

func TestEngine_promptCacheBreak_twoCompactRetry_events(t *testing.T) {
	t.Setenv(features.EnvPromptCacheBreak, "1")
	t.Setenv(features.EnvPromptCacheBreakTrimResend, "0")
	t.Setenv(features.EnvPromptCacheBreakAutoCompact, "1")
	var compactCalls, compactRetryEvents int
	turn := &failTwicePromptCacheBreakThenOK{t: t}
	e := New(context.Background(), &Config{
		Deps: query.Deps{Turn: turn},
		CompactExecutor: func(ctx context.Context, phase compact.RunPhase, transcriptJSON []byte) (string, []byte, error) {
			_ = ctx
			_ = phase
			compactCalls++
			if compactCalls == 1 {
				return "c1", []byte(`[{"role":"user","content":[{"type":"text","text":"seed1"}]}]`), nil
			}
			return "c2", []byte(`[{"role":"user","content":[{"type":"text","text":"seed2"}]}]`), nil
		},
	})
	e.Submit("hi")
	for {
		select {
		case ev := <-e.Events():
			if ev.Kind == EventKindPromptCacheBreakRecovery && ev.PhaseDetail == "compact_retry" {
				compactRetryEvents++
			}
			if ev.Kind == EventKindDone {
				if compactRetryEvents != 2 {
					t.Fatalf("compact_retry events: want 2, got %d", compactRetryEvents)
				}
				if compactCalls != 2 {
					t.Fatalf("CompactExecutor calls: want 2, got %d", compactCalls)
				}
				e.Wait()
				return
			}
			if ev.Kind == EventKindError {
				t.Fatalf("error: %v", ev.Err)
			}
		case <-time.After(4 * time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestEngine_promptCacheBreakSuggestCompact(t *testing.T) {
	t.Setenv(features.EnvPromptCacheBreak, "")
	t.Setenv(features.EnvPromptCacheBreakSuggestCompact, "true")
	var sawSuggest bool
	e := New(context.Background(), &Config{
		Deps: query.Deps{
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

func TestEngine_blockingLimit_ErrBlockingLimit(t *testing.T) {
	t.Setenv(features.EnvContextCollapse, "")
	t.Setenv(features.EnvReactiveCompact, "")
	t.Setenv(features.EnvDisableCompact, "")
	t.Setenv(features.EnvDisableAutoCompact, "")
	t.Setenv(features.EnvAutoCompact, "")
	t.Setenv(features.EnvBlockingLimitOverride, "1")
	e := New(context.Background(), &Config{
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				t.Fatal("StreamAssistant must not run when blocking limit trips first")
				return "", nil
			}),
		},
		Model:               "m",
		MaxTokens:           1024,
		ContextWindowTokens: 50_000,
	})
	e.Submit(strings.Repeat("z", 800))
	for {
		select {
		case ev := <-e.Events():
			if ev.Kind == EventKindError && errors.Is(ev.Err, query.ErrBlockingLimit) {
				e.Wait()
				return
			}
			if ev.Kind == EventKindDone {
				t.Fatal("unexpected Done before error")
			}
		case <-time.After(6 * time.Second):
			t.Fatal("timeout")
		}
	}
}

func TestEngine_restoredAutoCompact_consecutiveFailuresAndSnapshot(t *testing.T) {
	t.Setenv(features.EnvDisableCompact, "")
	t.Setenv(features.EnvDisableAutoCompact, "")
	t.Setenv(features.EnvAutoCompact, "")
	cf := 2
	e := New(context.Background(), &Config{
		InitialAutocompactConsecutiveFailures: 99,
		RestoredAutoCompactTracking: &compact.AutoCompactTracking{
			Compacted:           true,
			TurnCounter:         5,
			TurnID:              "autocompact:9",
			ConsecutiveFailures: &cf,
		},
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "ok", nil
			}),
		},
		Model: "m", MaxTokens: 64,
	})
	e.Submit("hi")
	drainEngineUntilDone(t, e.Events())
	e.Wait()
	if e.AutoCompactTrackingForPersistence() == nil {
		t.Fatal("expected snapshot after Submit")
	}
	data, err := compact.MarshalAutoCompactTrackingJSON(e.AutoCompactTrackingForPersistence())
	if err != nil {
		t.Fatal(err)
	}
	restored, err := compact.UnmarshalAutoCompactTrackingJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	e2 := New(context.Background(), &Config{
		RestoredAutoCompactTracking: restored,
		Deps: query.Deps{
			Assistant: query.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
				return "ok", nil
			}),
		},
		Model: "m", MaxTokens: 64,
	})
	if e2.autoCompactConsecutiveFailures != *restored.ConsecutiveFailures {
		t.Fatalf("engine consecutive failures want %d got %d", *restored.ConsecutiveFailures, e2.autoCompactConsecutiveFailures)
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
