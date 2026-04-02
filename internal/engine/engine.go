package engine

import (
	"context"
	"sync"
	"time"
)

// Engine coordinates one cancellable query turn stub (full query loop in internal/query later).
type Engine struct {
	ctx    context.Context
	cancel context.CancelFunc
	ch     chan EngineEvent
	wg     sync.WaitGroup
}

// NewEngine returns an engine whose lifetime is bounded by parent context or Cancel.
func NewEngine(parent context.Context) *Engine {
	ctx, cancel := context.WithCancel(parent)
	return &Engine{
		ctx:    ctx,
		cancel: cancel,
		ch:     make(chan EngineEvent, 64),
	}
}

// Events receives UserSubmit / AssistantText / Done for each Submit while not cancelled.
func (e *Engine) Events() <-chan EngineEvent {
	return e.ch
}

// Submit runs a stub assistant turn asynchronously (replace with real streamer in later commits).
func (e *Engine) Submit(userText string) {
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		if !e.trySend(EngineEvent{Kind: EventKindUserSubmit, UserText: userText}) {
			return
		}
		select {
		case <-e.ctx.Done():
			return
		case <-time.After(50 * time.Millisecond):
		}
		if !e.trySend(EngineEvent{Kind: EventKindAssistantText, AssistText: "stub"}) {
			return
		}
		e.trySend(EngineEvent{Kind: EventKindDone})
	}()
}

func (e *Engine) trySend(ev EngineEvent) bool {
	select {
	case <-e.ctx.Done():
		return false
	case e.ch <- ev:
		return true
	}
}

// Cancel stops in-flight Submit work (idempotent).
func (e *Engine) Cancel() {
	e.cancel()
}

// Wait blocks until all Submit goroutines finish after Cancel.
func (e *Engine) Wait() {
	e.wg.Wait()
}
