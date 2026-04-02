package engine

import (
	"context"
	"testing"
	"time"
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
