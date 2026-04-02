package engine

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/2456868764/rabbit-code/internal/compact"
	"github.com/2456868764/rabbit-code/internal/memdir"
	"github.com/2456868764/rabbit-code/internal/query"
	"github.com/2456868764/rabbit-code/internal/querydeps"
)

// Config configures optional streaming backend (nil Assistant keeps stub behavior).
type Config struct {
	Deps        querydeps.Deps
	Model       string
	MaxTokens   int
	StubDelay   time.Duration // for tests when Assistant is nil; zero uses default
	MemdirPaths []string      // optional: prepend session fragments to each Submit user text (P5.4.1)
	// CompactAdvisor, if set, runs after a successful turn loop to surface scheduling hints (P5.2.1 stub).
	CompactAdvisor func(st query.LoopState, transcriptJSONLen int) (autoCompact, reactiveCompact bool)
}

// Engine coordinates cancellable query turns (stub or real StreamAssistant / RunTurnLoop).
type Engine struct {
	ctx            context.Context
	cancel         context.CancelFunc
	ch             chan EngineEvent
	wg             sync.WaitGroup
	deps           querydeps.Deps
	model          string
	maxTokens      int
	stubDelay      time.Duration
	memdirPaths    []string
	compactAdvisor func(query.LoopState, int) (bool, bool)
}

// NewEngine is equivalent to New(parent, nil) (stub assistant).
func NewEngine(parent context.Context) *Engine {
	return New(parent, nil)
}

// New constructs an engine. Nil cfg or nil cfg.Assistant uses timed stub text.
// When Assistant is *querydeps.AnthropicAssistant and Turn is nil, Turn is promoted to the same pointer so tool_use streams work.
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
		deps := cfg.Deps
		if deps.Turn == nil {
			if aa, ok := deps.Assistant.(*querydeps.AnthropicAssistant); ok {
				deps.Turn = aa
			}
		}
		e.deps = deps
		if cfg.Model != "" {
			e.model = cfg.Model
		}
		if cfg.MaxTokens > 0 {
			e.maxTokens = cfg.MaxTokens
		}
		if cfg.StubDelay > 0 {
			e.stubDelay = cfg.StubDelay
		}
		e.memdirPaths = append([]string(nil), cfg.MemdirPaths...)
		e.compactAdvisor = cfg.CompactAdvisor
	}
	return e
}

// Events receives engine lifecycle events.
func (e *Engine) Events() <-chan EngineEvent {
	return e.ch
}

func (e *Engine) useQueryLoop() bool {
	return e.deps.Assistant != nil || e.deps.Turn != nil
}

// Submit runs one user turn: stub, single StreamAssistant call, or query.RunTurnLoop when assistant/turn is configured.
func (e *Engine) Submit(userText string) {
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		if !e.trySend(EngineEvent{Kind: EventKindUserSubmit, UserText: userText}) {
			return
		}
		if e.useQueryLoop() {
			e.runTurnLoop(userText)
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

func (e *Engine) applyMemdir(userText string) (resolved string, nFrag int, err error) {
	if len(e.memdirPaths) == 0 {
		return userText, 0, nil
	}
	frags, err := memdir.SessionFragmentsFromPaths(e.memdirPaths)
	if err != nil {
		return "", 0, err
	}
	if len(frags) == 0 {
		return userText, 0, nil
	}
	var b strings.Builder
	for _, f := range frags {
		b.WriteString(f)
		b.WriteString("\n\n")
	}
	b.WriteString(userText)
	return b.String(), len(frags), nil
}

func (e *Engine) runTurnLoop(userText string) {
	resolved, nFrag, err := e.applyMemdir(userText)
	if err != nil {
		e.trySend(EngineEvent{Kind: EventKindError, Err: err})
		return
	}
	if nFrag > 0 {
		if !e.trySend(EngineEvent{Kind: EventKindMemdirInject, MemdirFragmentCount: nFrag}) {
			return
		}
	}

	st := &query.LoopState{}
	d := query.LoopDriver{
		Deps: querydeps.Deps{
			Tools:     e.deps.Tools,
			Assistant: e.deps.Assistant,
			Turn:      e.deps.Turn,
		},
		Model:     e.model,
		MaxTokens: e.maxTokens,
		Observe: &query.LoopObservers{
			OnAssistantText: func(text string) {
				if text != "" {
					e.trySend(EngineEvent{Kind: EventKindAssistantText, AssistText: text})
				}
			},
			OnToolStart: func(name, id string, input []byte) {
				e.trySend(EngineEvent{
					Kind:          EventKindToolCallStart,
					ToolName:      name,
					ToolUseID:     id,
					ToolInputJSON: string(input),
				})
			},
			OnToolDone: func(name, id string, result []byte) {
				e.trySend(EngineEvent{
					Kind:           EventKindToolCallDone,
					ToolName:       name,
					ToolUseID:      id,
					ToolResultJSON: string(result),
				})
			},
		},
	}

	msgs, _, err := d.RunTurnLoop(e.ctx, st, resolved)
	if err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(e.ctx.Err(), context.Canceled) {
			return
		}
		st.HadStreamError = true
		e.trySend(EngineEvent{Kind: EventKindError, Err: err})
		return
	}

	if e.compactAdvisor != nil {
		auto, react := e.compactAdvisor(*st, len(msgs))
		if auto || react {
			phase := compact.RunIdle.Next(auto, react)
			e.trySend(EngineEvent{
				Kind:                   EventKindCompactSuggest,
				CompactPhase:           phase.String(),
				SuggestAutoCompact:     auto,
				SuggestReactiveCompact: react,
			})
		}
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

// Cancel stops in-flight Submit work (idempotent). In-flight HTTP streams should respect the same context when wired through RunTurnLoop.
func (e *Engine) Cancel() {
	e.cancel()
}

// Wait blocks until all Submit goroutines finish after Cancel.
func (e *Engine) Wait() {
	e.wg.Wait()
}
