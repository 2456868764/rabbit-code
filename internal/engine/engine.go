package engine

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/2456868764/rabbit-code/internal/querydeps"
)

// Config configures optional streaming backend (nil Assistant keeps stub behavior).
type Config struct {
	Assistant   querydeps.StreamAssistant
	Model       string
	MaxTokens   int
	StubDelay   time.Duration // for tests when Assistant is nil; zero uses default
}

// Engine coordinates cancellable query turns (stub or real StreamAssistant).
type Engine struct {
	ctx       context.Context
	cancel    context.CancelFunc
	ch        chan EngineEvent
	wg        sync.WaitGroup
	assistant querydeps.StreamAssistant
	model     string
	maxTokens int
	stubDelay time.Duration
}

// NewEngine is equivalent to New(parent, nil) (stub assistant).
func NewEngine(parent context.Context) *Engine {
	return New(parent, nil)
}

// New constructs an engine. Nil cfg or nil cfg.Assistant uses timed stub text.
func New(parent context.Context, cfg *Config) *Engine {
	ctx, cancel := context.WithCancel(parent)
	e := &Engine{
		ctx:       ctx,
		cancel:    cancel,
		ch:        make(chan EngineEvent, 64),
		model:     "claude-3-5-haiku-20241022",
		maxTokens: 1024,
		stubDelay: 50 * time.Millisecond,
	}
	if cfg != nil {
		e.assistant = cfg.Assistant
		if cfg.Model != "" {
			e.model = cfg.Model
		}
		if cfg.MaxTokens > 0 {
			e.maxTokens = cfg.MaxTokens
		}
		if cfg.StubDelay > 0 {
			e.stubDelay = cfg.StubDelay
		}
	}
	return e
}

// Events receives engine lifecycle events.
func (e *Engine) Events() <-chan EngineEvent {
	return e.ch
}

// Submit runs one user turn: optional StreamAssistant, else stub.
func (e *Engine) Submit(userText string) {
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		if !e.trySend(EngineEvent{Kind: EventKindUserSubmit, UserText: userText}) {
			return
		}
		if e.assistant != nil {
			e.runAssistant(userText)
			return
		}
		select {
		case <-e.ctx.Done():
			return
		case <-time.After(e.stubDelay):
		}
		if !e.trySend(EngineEvent{Kind: EventKindAssistantText, AssistText: "stub"}) {
			return
		}
		e.trySend(EngineEvent{Kind: EventKindDone})
	}()
}

func (e *Engine) runAssistant(userText string) {
	payload, err := json.Marshal([]map[string]any{
		{"role": "user", "content": []map[string]string{{"type": "text", "text": userText}}},
	})
	if err != nil {
		e.trySend(EngineEvent{Kind: EventKindError, Err: err})
		return
	}
	text, err := e.assistant.StreamAssistant(e.ctx, e.model, e.maxTokens, payload)
	if err != nil {
		e.trySend(EngineEvent{Kind: EventKindError, Err: err})
		return
	}
	if !e.trySend(EngineEvent{Kind: EventKindAssistantText, AssistText: text}) {
		return
	}
	e.trySend(EngineEvent{Kind: EventKindDone})
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
