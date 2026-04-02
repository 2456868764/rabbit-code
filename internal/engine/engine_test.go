package engine

import (
	"context"
	"errors"
	"testing"
	"time"

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
		Assistant: querydeps.StreamAssistantFunc(func(ctx context.Context, model string, maxTokens int, messagesJSON []byte) (string, error) {
			if model != "m" || maxTokens != 16 {
				t.Fatalf("model=%q max=%d", model, maxTokens)
			}
			return "assistant-out", nil
		}),
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
		Assistant: querydeps.StreamAssistantFunc(func(context.Context, string, int, []byte) (string, error) {
			return "", errors.New("stream err")
		}),
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
