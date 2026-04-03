package engine

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/2456868764/rabbit-code/internal/anthropic"
	"github.com/2456868764/rabbit-code/internal/compact"
	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/memdir"
	"github.com/2456868764/rabbit-code/internal/query"
	"github.com/2456868764/rabbit-code/internal/querydeps"
)

// StopHookFunc runs after each Submit’s RunTurnLoop attempt finishes (success or failure). Hooks run in slice order; legacy StopHook is appended after StopHooks (P5.1.4).
type StopHookFunc func(ctx context.Context, st query.LoopState, err error)

// RecoverStrategy returns true to run exactly one additional RunTurnLoop after a failure (P5.1.3).
type RecoverStrategy func(ctx context.Context, st query.LoopState, err error) bool

// CompactExecutor runs after a compact suggest when set (P5.2.1 stub / closure).
type CompactExecutor func(ctx context.Context, phase compact.RunPhase, transcriptJSON []byte) (summary string, err error)

// Config configures optional streaming backend (nil Assistant keeps stub behavior).
type Config struct {
	Deps        querydeps.Deps
	Model       string
	MaxTokens   int
	StubDelay   time.Duration // for tests when Assistant is nil; zero uses default
	MemdirPaths []string      // optional: prepend session fragments to each Submit user text (P5.4.1)
	// MaxAssistantTurns if > 0 sets query.LoopState.MaxTurns for each Submit (caps assistant API rounds).
	MaxAssistantTurns int
	// SuggestCompactOnRecoverableError emits EventKindCompactSuggest (auto) before EventKindError when the failure is RecoverableCompact (P5.1.3 hint).
	SuggestCompactOnRecoverableError bool
	// CompactAdvisor, if set, runs after a successful turn loop to surface scheduling hints (P5.2.1 stub).
	CompactAdvisor func(st query.LoopState, transcriptJSON []byte) (autoCompact, reactiveCompact bool)
	// CompactExecutor, if set, runs after each CompactSuggest from CompactAdvisor; emits EventKindCompactResult (P5.2.1).
	CompactExecutor CompactExecutor
	// StopHooks run after RunTurnLoop finishes for the Submit (see StopHookFunc).
	StopHooks []StopHookFunc
	// StopHook is equivalent to appending one element to StopHooks (backward compatible).
	StopHook StopHookFunc
	// RecoverStrategy enables a second RunTurnLoop attempt when it returns true on the first failure.
	RecoverStrategy RecoverStrategy
	// OrphanPermissionAdvisor, if set, runs after a successful loop; emit EventKindOrphanPermission when ok (P5.3.3 stub).
	OrphanPermissionAdvisor func(st query.LoopState) (orphanToolUseID string, ok bool)
	// TemplateDir if set overrides RABBIT_CODE_TEMPLATE_DIR for loading <name>.md when TEMPLATES is on (P5.F.7).
	TemplateDir string
}

// Engine coordinates cancellable query turns (stub or real StreamAssistant / RunTurnLoop).
type Engine struct {
	ctx                              context.Context
	cancel                           context.CancelFunc
	ch                               chan EngineEvent
	wg                               sync.WaitGroup
	deps                             querydeps.Deps
	model                            string
	maxTokens                        int
	stubDelay                        time.Duration
	memdirPaths                      []string
	compactAdvisor                   func(query.LoopState, []byte) (bool, bool)
	compactExecutor                  CompactExecutor
	stopHooks                        []StopHookFunc
	recoverStrategy                  RecoverStrategy
	orphanPermissionAdvisor          func(query.LoopState) (string, bool)
	maxAssistantTurns                int
	suggestCompactOnRecoverableError bool
	templateDir                      string
	cacheBreakSeen                   int32 // atomic: prompt-cache break callback ran this Submit
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
		e.compactExecutor = cfg.CompactExecutor
		e.stopHooks = append([]StopHookFunc(nil), cfg.StopHooks...)
		if cfg.StopHook != nil {
			e.stopHooks = append(e.stopHooks, cfg.StopHook)
		}
		e.recoverStrategy = cfg.RecoverStrategy
		e.orphanPermissionAdvisor = cfg.OrphanPermissionAdvisor
		if cfg.MaxAssistantTurns > 0 {
			e.maxAssistantTurns = cfg.MaxAssistantTurns
		}
		e.suggestCompactOnRecoverableError = cfg.SuggestCompactOnRecoverableError
		e.templateDir = strings.TrimSpace(cfg.TemplateDir)
	}
	return e
}

func (e *Engine) templateAppendixDir() string {
	if e.templateDir != "" {
		return e.templateDir
	}
	return features.TemplateMarkdownDir()
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
		modeTags := query.FormatPhase5HeadlessModeTags(query.Phase5UserTextFlags{
			ContextCollapse: features.ContextCollapseEnabled(),
			Ultrathink:      features.UltrathinkEnabled(),
			Ultraplan:       features.UltraplanEnabled(),
			SessionRestore:  features.SessionRestoreEnabled(),
		})
		if !e.trySend(EngineEvent{Kind: EventKindUserSubmit, UserText: userText, PhaseDetail: modeTags}) {
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

func (e *Engine) applyMemdir(userText string) (resolved string, nFrag int, injectRawBytes int, err error) {
	if len(e.memdirPaths) == 0 {
		return userText, 0, 0, nil
	}
	frags, injectRawBytes, err := memdir.SessionFragmentsFromPaths(e.memdirPaths)
	if err != nil {
		return "", 0, 0, err
	}
	if len(frags) == 0 {
		return userText, 0, injectRawBytes, nil
	}
	var b strings.Builder
	for _, f := range frags {
		b.WriteString(f)
		b.WriteString("\n\n")
	}
	b.WriteString(userText)
	return b.String(), len(frags), injectRawBytes, nil
}

func (e *Engine) invokeStopHooks(st *query.LoopState, loopErr error) {
	for _, h := range e.stopHooks {
		if h != nil {
			h(e.ctx, *st, loopErr)
		}
	}
}

func (e *Engine) loopObservers() *query.LoopObservers {
	return &query.LoopObservers{
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
		OnToolError: func(name, id string, err error) {
			e.trySend(EngineEvent{
				Kind:      EventKindToolCallFailed,
				ToolName:  name,
				ToolUseID: id,
				Err:       err,
			})
			if oid, ok := querydeps.OrphanToolUseID(err); ok && oid != "" {
				e.trySend(EngineEvent{
					Kind:            EventKindOrphanPermission,
					OrphanToolUseID: oid,
				})
			}
		},
		OnHistorySnip: func(before, after, rounds int) {
			e.trySend(EngineEvent{
				Kind:         EventKindHistorySnipApplied,
				PhaseDetail:  fmt.Sprintf("rounds=%d", rounds),
				PhaseAuxInt:  before,
				PhaseAuxInt2: after,
			})
		},
		OnSnipCompact: func(before, after, rounds int) {
			e.trySend(EngineEvent{
				Kind:         EventKindSnipCompactApplied,
				PhaseDetail:  fmt.Sprintf("rounds=%d", rounds),
				PhaseAuxInt:  before,
				PhaseAuxInt2: after,
			})
		},
	}
}

func (e *Engine) runTurnLoop(userText string) {
	st := &query.LoopState{}
	var loopErr error
	defer func() { e.invokeStopHooks(st, loopErr) }()

	resolved, nFrag, injectRaw, err := e.applyMemdir(userText)
	if err != nil {
		loopErr = err
		e.trySend(EngineEvent{Kind: EventKindError, Err: err})
		return
	}
	if nFrag > 0 {
		if !e.trySend(EngineEvent{Kind: EventKindMemdirInject, MemdirFragmentCount: nFrag}) {
			return
		}
	}

	if dir := e.templateAppendixDir(); dir != "" {
		names := features.TemplateNames()
		if len(names) > 0 {
			app, err := query.LoadTemplateMarkdownAppendix(dir, names)
			if err != nil {
				loopErr = err
				e.trySend(EngineEvent{Kind: EventKindError, Err: err})
				return
			}
			if app != "" {
				resolved += app
			}
		}
	}

	if maxA := features.TokenBudgetMaxAttachmentBytes(); maxA > 0 && injectRaw > maxA {
		loopErr = ErrTokenBudgetExceeded
		e.trySend(EngineEvent{Kind: EventKindError, Err: loopErr})
		return
	}
	if maxB := features.TokenBudgetMaxInputBytes(); maxB > 0 && len(resolved) > maxB {
		loopErr = ErrTokenBudgetExceeded
		e.trySend(EngineEvent{Kind: EventKindError, Err: loopErr})
		return
	}
	if maxT := features.TokenBudgetMaxInputTokens(); maxT > 0 && query.EstimateUTF8BytesAsTokens(resolved) > maxT {
		loopErr = ErrTokenBudgetExceeded
		e.trySend(EngineEvent{Kind: EventKindError, Err: loopErr})
		return
	}

	if features.BreakCacheCommandEnabled() {
		e.trySend(EngineEvent{Kind: EventKindBreakCacheCommand, PhaseDetail: "submit"})
	}
	if features.TemplatesEnabled() {
		names := features.TemplateNames()
		e.trySend(EngineEvent{Kind: EventKindTemplatesActive, PhaseDetail: strings.Join(names, ",")})
	}
	if features.CachedMicrocompactEnabled() {
		e.trySend(EngineEvent{Kind: EventKindCachedMicrocompactActive, PhaseDetail: "api_body_flags_deferred"})
	}

	resolved = query.ApplyPhase5UserTextHints(resolved, query.Phase5UserTextFlags{
		ContextCollapse: features.ContextCollapseEnabled(),
		Ultrathink:      features.UltrathinkEnabled(),
		Ultraplan:       features.UltraplanEnabled(),
		SessionRestore:  features.SessionRestoreEnabled(),
	})

	atomic.StoreInt32(&e.cacheBreakSeen, 0)
	ctxLoop := e.ctx
	if features.PromptCacheBreakDetectionEnabled() || features.PromptCacheBreakSuggestCompactEnabled() {
		ctxLoop = querydeps.ContextWithOnPromptCacheBreak(e.ctx, func() {
			atomic.StoreInt32(&e.cacheBreakSeen, 1)
			if features.PromptCacheBreakDetectionEnabled() {
				e.trySend(EngineEvent{Kind: EventKindPromptCacheBreakDetected, PhaseDetail: "sse"})
			}
		})
	}

	maxAttempts := 1
	if e.recoverStrategy != nil {
		maxAttempts = 2
	}

	var msgs json.RawMessage
	succeeded := false
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt == 0 {
			if e.maxAssistantTurns > 0 {
				st.MaxTurns = e.maxAssistantTurns
			}
		} else {
			resetLoopStateForRetryAttempt(st)
		}

		d := query.LoopDriver{
			Deps: querydeps.Deps{
				Tools:     e.deps.Tools,
				Assistant: e.deps.Assistant,
				Turn:      e.deps.Turn,
			},
			Model:                e.model,
			MaxTokens:            e.maxTokens,
			Observe:              e.loopObservers(),
			HistorySnipMaxBytes:  features.HistorySnipMaxBytes(),
			HistorySnipMaxRounds: features.HistorySnipMaxRounds(),
			SnipCompactMaxBytes:  features.SnipCompactMaxBytes(),
			SnipCompactMaxRounds: features.SnipCompactMaxRounds(),
		}

		var runErr error
		msgs, _, runErr = d.RunTurnLoop(ctxLoop, st, resolved)
		if runErr == nil {
			succeeded = true
			loopErr = nil
			break
		}
		loopErr = runErr
		if errors.Is(runErr, context.Canceled) || errors.Is(e.ctx.Err(), context.Canceled) {
			return
		}
		st.HadStreamError = true
		kind, rec := classifyAnthropicError(runErr)
		st.LastAPIErrorKind = kind
		if rec {
			st.RecoveryAttempts++
			if st.RecoveryPhase == query.RecoveryNone {
				st.RecoveryPhase = query.RecoveryPendingCompact
			}
		}
		if rec && e.suggestCompactOnRecoverableError {
			ph := compact.RunIdle.Next(true, false)
			e.trySend(EngineEvent{
				Kind:               EventKindCompactSuggest,
				CompactPhase:       ph.String(),
				SuggestAutoCompact: true,
			})
			if e.compactExecutor != nil {
				sum, exErr := e.compactExecutor(e.ctx, ph, msgs)
				e.trySend(EngineEvent{
					Kind:           EventKindCompactResult,
					CompactPhase:   ph.String(),
					CompactSummary: sum,
					Err:            exErr,
				})
				if exErr == nil {
					query.RecordLoopContinue(st, query.LoopContinue{Reason: query.ContinueReasonReactiveCompactRetry})
				}
			}
		}
		willRetry := attempt+1 < maxAttempts && e.recoverStrategy != nil && e.recoverStrategy(e.ctx, *st, runErr)
		if willRetry {
			if kind == string(anthropic.KindMaxOutputTokens) {
				st.MaxOutputTokensRecoveryCount++
				query.RecordLoopContinue(st, query.LoopContinue{
					Reason:  query.ContinueReasonMaxOutputTokensRecovery,
					Attempt: st.MaxOutputTokensRecoveryCount,
				})
			} else {
				query.RecordLoopContinue(st, query.LoopContinue{Reason: query.ContinueReasonSubmitRecoverRetry})
			}
			continue
		}
		e.trySend(EngineEvent{
			Kind:               EventKindError,
			Err:                runErr,
			APIErrorKind:       kind,
			RecoverableCompact: rec,
		})
		return
	}

	if !succeeded {
		return
	}

	if features.PromptCacheBreakSuggestCompactEnabled() && atomic.SwapInt32(&e.cacheBreakSeen, 0) != 0 {
		ph := compact.RunIdle.Next(false, true)
		e.trySend(EngineEvent{
			Kind:                   EventKindCompactSuggest,
			CompactPhase:           ph.String(),
			SuggestReactiveCompact: true,
		})
		if e.compactExecutor != nil {
			sum, exErr := e.compactExecutor(e.ctx, ph, msgs)
			e.trySend(EngineEvent{
				Kind:           EventKindCompactResult,
				CompactPhase:   ph.String(),
				CompactSummary: sum,
				Err:            exErr,
			})
			if exErr == nil {
				query.RecordLoopContinue(st, query.LoopContinue{Reason: query.ContinueReasonReactiveCompactRetry})
			}
		}
	}

	if e.orphanPermissionAdvisor != nil {
		if id, ok := e.orphanPermissionAdvisor(*st); ok && id != "" {
			e.trySend(EngineEvent{
				Kind:            EventKindOrphanPermission,
				OrphanToolUseID: id,
			})
		}
	}

	auto, react := false, false
	if e.compactAdvisor != nil {
		auto, react = e.compactAdvisor(*st, msgs)
	}
	if query.ReactiveCompactByTranscript(msgs, features.ReactiveCompactMinTranscriptBytes(), features.ReactiveCompactMinEstimatedTokens()) {
		react = true
	}
	if auto || react {
		phase := compact.RunIdle.Next(auto, react)
		e.trySend(EngineEvent{
			Kind:                   EventKindCompactSuggest,
			CompactPhase:           phase.String(),
			SuggestAutoCompact:     auto,
			SuggestReactiveCompact: react,
		})
		if e.compactExecutor != nil {
			sum, exErr := e.compactExecutor(e.ctx, phase, msgs)
			e.trySend(EngineEvent{
				Kind:           EventKindCompactResult,
				CompactPhase:   phase.String(),
				CompactSummary: sum,
				Err:            exErr,
			})
			if exErr == nil {
				if react {
					query.RecordLoopContinue(st, query.LoopContinue{Reason: query.ContinueReasonReactiveCompactRetry})
				} else if auto {
					query.RecordLoopContinue(st, query.LoopContinue{Reason: query.ContinueReasonAutoCompactExecuted})
				}
			}
		}
	}

	e.trySend(EngineEvent{Kind: EventKindDone, LoopTurnCount: st.TurnCount})
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
