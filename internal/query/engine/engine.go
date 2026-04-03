package engine

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/memdir"
	"github.com/2456868764/rabbit-code/internal/query"
	"github.com/2456868764/rabbit-code/internal/query/querydeps"
	anthropic "github.com/2456868764/rabbit-code/internal/services/api"
	"github.com/2456868764/rabbit-code/internal/services/compact"
)

// StopHookFunc runs after each Submit’s RunTurnLoop attempt finishes (success or failure). Hooks run in slice order; legacy StopHook is appended after StopHooks (P5.1.4).
type StopHookFunc func(ctx context.Context, st query.LoopState, err error)

// RecoverStrategy returns true to run exactly one additional RunTurnLoop after a failure (P5.1.3).
type RecoverStrategy func(ctx context.Context, st query.LoopState, err error) bool

// CompactExecutor runs after a compact suggest when set (P5.2.1 stub / closure).
// When nextTranscriptJSON is non-empty, RecoverStrategy retry seeds RunTurnLoopFromMessages with it (H6 reactive compact path).
type CompactExecutor func(ctx context.Context, phase compact.RunPhase, transcriptJSON []byte) (summary string, nextTranscriptJSON []byte, err error)

// SessionMemoryCompact runs before legacy CompactExecutor when proactive auto compact is scheduled (autoCompact.ts trySessionMemoryCompaction).
// If ok and replacement is non-empty, the engine emits compact suggest/result and skips the legacy auto executor for that wave.
type SessionMemoryCompact func(ctx context.Context, agentID, model string, autoCompactThreshold int, transcriptJSON json.RawMessage) (replacement json.RawMessage, ok bool, err error)

// PostCompactCleanup mirrors runPostCompactCleanup side effects (postCompactCleanup.ts); mainThreadCompact matches isMainThreadCompact.
type PostCompactCleanup func(ctx context.Context, querySource, agentID string, mainThreadCompact bool)

// ContextCollapseDrain trims transcript on recoverable prompt_too_long when CONTEXT_COLLAPSE is on; committed feeds LoopContinue (H6).
type ContextCollapseDrain func(ctx context.Context, st *query.LoopState, transcriptJSON json.RawMessage) (trimmed json.RawMessage, committed int, ok bool)

// StopHookBlockingContinue returns true to run another RunTurnLoop after success (query.ts stop_hook_blocking).
type StopHookBlockingContinue func(ctx context.Context, st query.LoopState) bool

// TokenBudgetContinueAfterTurn returns true to run another RunTurnLoop when TOKEN_BUDGET is on (query.ts token_budget_continuation).
type TokenBudgetContinueAfterTurn func(ctx context.Context, st query.LoopState, transcriptJSON json.RawMessage) bool

// StopHookAfterTurnResult is the headless analogue of query/stopHooks.ts aggregate after handleStopHooks (H6).
type StopHookAfterTurnResult struct {
	// BlockingContinue requests another RunTurnLoop (query.ts stop_hook_blocking).
	BlockingContinue bool
	// PreventContinuation ends Submit like query.ts stop_hook_prevented (skips token budget and further turns).
	PreventContinuation bool
	// StopReason optional; default PhaseDetail on Done is stop_hook_prevented.
	StopReason string
}

// StopHookAfterTurnFunc runs after a successful RunTurnLoop wave, before StopHookBlockingContinue / token budget (query.ts order).
type StopHookAfterTurnFunc func(ctx context.Context, st query.LoopState, transcriptJSON json.RawMessage) StopHookAfterTurnResult

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
	// ContextWindowTokens if > 0 overrides RABBIT_CODE_CONTEXT_WINDOW_TOKENS / model default for proactive autocompact threshold (H2 / autoCompact.ts).
	ContextWindowTokens int
	// ContextCollapseDrain optional trim before compact on PTL when RABBIT_CODE_CONTEXT_COLLAPSE is on (H6).
	ContextCollapseDrain ContextCollapseDrain
	// StopHookBlockingContinue optional second RunTurnLoop after success (H6).
	StopHookBlockingContinue StopHookBlockingContinue
	// TokenBudgetContinueAfterTurn optional extra RunTurnLoop when RABBIT_CODE_TOKEN_BUDGET is on (H6).
	TokenBudgetContinueAfterTurn TokenBudgetContinueAfterTurn
	// AgentID optional mirror of query.ts toolUseContext.agentId (H6).
	AgentID string
	// NonInteractive mirrors toolUseContext.options.isNonInteractiveSession (H6).
	NonInteractive bool
	// SessionID optional ToolUseContextMirror / analytics (H6).
	SessionID string
	// Debug mirrors toolUseContext.options.debug (H6).
	Debug bool
	// QuerySource optional fork id for shouldAutoCompact gates (H2 / autoCompact.ts).
	QuerySource string
	// StopHooksAfterSuccessfulTurn run after each successful turn-loop wave (see StopHookAfterTurnFunc).
	StopHooksAfterSuccessfulTurn []StopHookAfterTurnFunc
	// SessionMemoryCompact optional first step before legacy auto compact (H3 / autoCompact.ts).
	SessionMemoryCompact SessionMemoryCompact
	// PostCompactCleanup optional hook after any successful compact executor / session-memory compact (H3).
	PostCompactCleanup PostCompactCleanup
	// MicrocompactEditBuffer optional; reset on successful compact and wired to AnthropicAssistant when possible (H4).
	MicrocompactEditBuffer *compact.MicrocompactEditBuffer
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
	contextWindowTokens              int
	contextCollapseDrain             ContextCollapseDrain
	stopHookBlockingContinue         StopHookBlockingContinue
	tokenBudgetContinueAfterTurn     TokenBudgetContinueAfterTurn
	agentID                          string
	nonInteractive                   bool
	sessionID                        string
	debug                            bool
	querySource                      string
	stopHooksAfterSuccessfulTurn     []StopHookAfterTurnFunc
	sessionMemoryCompact             SessionMemoryCompact
	postCompactCleanup               PostCompactCleanup
	microcompactEditBuffer           *compact.MicrocompactEditBuffer
	cacheBreakSeen                   int32 // atomic: prompt-cache break callback ran this Submit
	// autoCompactConsecutiveFailures counts failed proactive auto compact executor runs across Submits (H3 / autoCompact.ts);
	// mirrored onto st.AutoCompactTracking.ConsecutiveFailures when st != nil.
	autoCompactConsecutiveFailures int
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
		if cfg.ContextWindowTokens > 0 {
			e.contextWindowTokens = cfg.ContextWindowTokens
		}
		e.contextCollapseDrain = cfg.ContextCollapseDrain
		e.stopHookBlockingContinue = cfg.StopHookBlockingContinue
		e.tokenBudgetContinueAfterTurn = cfg.TokenBudgetContinueAfterTurn
		e.agentID = strings.TrimSpace(cfg.AgentID)
		e.nonInteractive = cfg.NonInteractive
		e.sessionID = strings.TrimSpace(cfg.SessionID)
		e.debug = cfg.Debug
		e.querySource = strings.TrimSpace(cfg.QuerySource)
		e.stopHooksAfterSuccessfulTurn = append([]StopHookAfterTurnFunc(nil), cfg.StopHooksAfterSuccessfulTurn...)
		e.sessionMemoryCompact = cfg.SessionMemoryCompact
		e.postCompactCleanup = cfg.PostCompactCleanup
		e.microcompactEditBuffer = cfg.MicrocompactEditBuffer
		if aa, ok := e.deps.Assistant.(*querydeps.AnthropicAssistant); ok && cfg.MicrocompactEditBuffer != nil {
			aa.MicrocompactBuffer = cfg.MicrocompactEditBuffer
		}
		if aa, ok := e.deps.Turn.(*querydeps.AnthropicAssistant); ok && cfg.MicrocompactEditBuffer != nil {
			aa.MicrocompactBuffer = cfg.MicrocompactEditBuffer
		}
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
		modeTags := query.FormatHeadlessModeTags(query.UserTextHintFlags{
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
		OnPromptCacheBreakRecovery: func(phase string) {
			e.trySend(EngineEvent{Kind: EventKindPromptCacheBreakRecovery, PhaseDetail: phase})
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
		e.trySend(EngineEvent{Kind: EventKindCachedMicrocompactActive, PhaseDetail: anthropic.BetaCachedMicrocompactBody})
	}

	resolved = query.ApplyUserTextHints(resolved, query.UserTextHintFlags{
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
	for round := 0; round < maxSubmitContinuationRounds; round++ {
		var subErr error
		msgs, succeeded, subErr = e.executeRunTurnLoopAttempts(ctxLoop, st, resolved, maxAttempts)
		if !succeeded {
			loopErr = subErr
			return
		}
		loopErr = nil
		var stopPrevent bool
		var stopReason string
		var blockFromAfterTurn bool
		for _, h := range e.stopHooksAfterSuccessfulTurn {
			if h == nil {
				continue
			}
			r := h(e.ctx, *st, msgs)
			if r.PreventContinuation {
				stopPrevent = true
				stopReason = r.StopReason
				break
			}
			if r.BlockingContinue {
				blockFromAfterTurn = true
			}
		}
		if stopPrevent {
			if stopReason == "" {
				stopReason = query.ContinueReasonStopHookPrevented
			}
			query.RecordLoopContinue(st, query.LoopContinue{Reason: query.ContinueReasonStopHookPrevented})
			e.trySend(EngineEvent{Kind: EventKindDone, LoopTurnCount: st.TurnCount, PhaseDetail: stopReason})
			return
		}
		needStopHookBlock := blockFromAfterTurn || (e.stopHookBlockingContinue != nil && e.stopHookBlockingContinue(e.ctx, *st))
		if needStopHookBlock {
			query.RecordLoopContinue(st, query.LoopContinue{Reason: query.ContinueReasonStopHookBlocking})
			PrepareLoopStateForStopHookBlockingContinuation(st)
			continue
		}
		if features.TokenBudgetEnabled() && e.tokenBudgetContinueAfterTurn != nil && e.tokenBudgetContinueAfterTurn(e.ctx, *st, msgs) {
			query.RecordLoopContinue(st, query.LoopContinue{Reason: query.ContinueReasonTokenBudgetContinuation})
			PrepareLoopStateForTokenBudgetContinuation(st)
			continue
		}
		break
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
			execPh := compact.ExecutorPhaseAfterSchedule(ph)
			sum, _, exErr := e.compactExecutor(e.ctx, execPh, msgs)
			resPh := compact.ResultPhaseAfterCompactExecutor(execPh, exErr)
			e.trySend(EngineEvent{
				Kind:           EventKindCompactResult,
				CompactPhase:   resPh.String(),
				CompactSummary: sum,
				Err:            exErr,
			})
			if exErr == nil {
				query.RecordLoopContinue(st, query.LoopContinue{Reason: query.ContinueReasonReactiveCompactRetry})
				st.HasAttemptedReactiveCompact = true
				e.afterCompactSuccess(st)
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
	cw := e.contextWindowTokens
	if cw <= 0 {
		cw = features.ContextWindowTokensForModel(e.model)
	}
	cw = features.ApplyAutoCompactWindowCap(cw)
	if !e.autoCompactCircuitTripped() &&
		query.ProactiveAutoCompactSuggestedWithSource(msgs, e.model, e.maxTokens, cw, 0, st.ToolUseContext.QuerySource) {
		auto = true
	}
	if query.TranscriptReactiveCompactSuggested(st, msgs, features.ReactiveCompactMinTranscriptBytes(), features.ReactiveCompactMinEstimatedTokens()) {
		react = true
	}

	if auto && e.sessionMemoryCompact != nil {
		th := query.AutoCompactThresholdForProactive(e.model, e.maxTokens, cw)
		if th > 0 {
			next, ok, smErr := e.sessionMemoryCompact(e.ctx, st.ToolUseContext.AgentID, e.model, th, msgs)
			switch {
			case smErr != nil:
				// Legacy auto path may still run.
			case ok && len(bytes.TrimSpace(next)) > 0:
				msgs = json.RawMessage(append([]byte(nil), next...))
				st.SetMessagesJSON(msgs)
				ph := compact.RunIdle.Next(true, false)
				execPh := compact.ExecutorPhaseAfterSchedule(ph)
				e.trySend(EngineEvent{
					Kind:               EventKindCompactSuggest,
					CompactPhase:       ph.String(),
					SuggestAutoCompact: true,
				})
				resPh := compact.ResultPhaseAfterCompactExecutor(execPh, nil)
				e.noteAutoCompactExecutorOutcome(st, true, nil)
				e.trySend(EngineEvent{
					Kind:           EventKindCompactResult,
					CompactPhase:   resPh.String(),
					CompactSummary: "session_memory_compact",
					Err:            nil,
				})
				query.RecordLoopContinue(st, query.LoopContinue{Reason: query.ContinueReasonAutoCompactExecuted})
				auto = false
				react = query.TranscriptReactiveCompactSuggested(st, msgs, features.ReactiveCompactMinTranscriptBytes(), features.ReactiveCompactMinEstimatedTokens())
				e.afterCompactSuccess(st)
			}
		}
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
			execPh := compact.ExecutorPhaseAfterSchedule(phase)
			sum, _, exErr := e.compactExecutor(e.ctx, execPh, msgs)
			resPh := compact.ResultPhaseAfterCompactExecutor(execPh, exErr)
			e.noteAutoCompactExecutorOutcome(st, auto, exErr)
			e.trySend(EngineEvent{
				Kind:           EventKindCompactResult,
				CompactPhase:   resPh.String(),
				CompactSummary: sum,
				Err:            exErr,
			})
			if exErr == nil {
				if react {
					query.RecordLoopContinue(st, query.LoopContinue{Reason: query.ContinueReasonReactiveCompactRetry})
					st.HasAttemptedReactiveCompact = true
				} else if auto {
					query.RecordLoopContinue(st, query.LoopContinue{Reason: query.ContinueReasonAutoCompactExecuted})
				}
				e.afterCompactSuccess(st)
			}
		}
	}

	e.trySend(EngineEvent{Kind: EventKindDone, LoopTurnCount: st.TurnCount})
}

func (e *Engine) effectiveQuerySource(st *query.LoopState) string {
	if st != nil {
		if s := strings.TrimSpace(st.ToolUseContext.QuerySource); s != "" {
			return s
		}
	}
	return e.querySource
}

func (e *Engine) afterCompactSuccess(st *query.LoopState) {
	compact.ResetMicrocompactStateIfAny(e.microcompactEditBuffer)
	if e.postCompactCleanup != nil {
		src := e.effectiveQuerySource(st)
		e.postCompactCleanup(e.ctx, src, e.agentID, compact.IsMainThreadPostCompactSource(src))
	}
}

func (e *Engine) autoCompactCircuitTripped() bool {
	return e.autoCompactConsecutiveFailures >= query.MaxConsecutiveAutocompactFailures
}

// noteAutoCompactExecutorOutcome updates the session-level circuit when the compact suggest included proactive auto.
func (e *Engine) noteAutoCompactExecutorOutcome(st *query.LoopState, autoInSuggest bool, err error) {
	if !autoInSuggest {
		return
	}
	if err != nil {
		e.autoCompactConsecutiveFailures++
	} else {
		e.autoCompactConsecutiveFailures = 0
	}
	query.MirrorAutocompactConsecutiveFailures(st, e.autoCompactConsecutiveFailures)
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
