package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/memdir"
	"github.com/2456868764/rabbit-code/internal/query"
	anthropic "github.com/2456868764/rabbit-code/internal/services/api"
	"github.com/2456868764/rabbit-code/internal/services/compact"
	"github.com/2456868764/rabbit-code/internal/utils/processuserinput"
	"github.com/2456868764/rabbit-code/internal/utils/thinking"
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

// AfterSessionMemoryCompactSuccess runs when session-memory compaction successfully replaces the transcript, before compact suggest/result events (autoCompact.ts notifyCompaction / setLastSummarizedMessageId analogue).
type AfterSessionMemoryCompactSuccess func(ctx context.Context, querySource, agentID string)

// PostCompactCleanup optional extra hook after compact (postCompactCleanup.ts); runs after compact.RunPostCompactCleanup when both are set.
// mainThreadCompact matches isMainThreadCompact.
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
	Deps        query.Deps
	Model       string
	MaxTokens   int
	StubDelay   time.Duration // for tests when Assistant is nil; zero uses default
	MemdirPaths []string      // optional: explicit paths prepended to each Submit (P5.4.1)
	// MemdirMemoryDir when set triggers FindRelevantMemories per Submit (recursive scan + heuristic or LLM; H8).
	MemdirMemoryDir string
	// MemdirProjectRoot seeds memdir.ResolveAutoMemDir; empty uses cwd at engine init.
	MemdirProjectRoot string
	// MemdirTrustedAutoMemoryDirectory is autoMemoryDirectory from trusted settings only — same layers as
	// paths.ts getAutoMemPathSetting (see config.LoadTrustedAutoMemoryDirectory doc). Never from project JSON.
	MemdirTrustedAutoMemoryDirectory string
	// InitialSettings optional merged settings (e.g. config.LoadMerged); gates auto memdir via autoMemoryEnabled like paths.ts getInitialSettings.
	InitialSettings map[string]interface{}
	// MemdirRecentTools is passed into LLM memdir selection (suppress tool-doc memories; H8).
	MemdirRecentTools []string
	// MemdirTextComplete optional override for LLM selection (tests); default uses Anthropic client when mode is llm.
	MemdirTextComplete memdir.TextCompleteFunc
	// MemdirRelevanceModeOverride when non-empty overrides RABBIT_CODE_MEMDIR_RELEVANCE_MODE ("heuristic"|"llm").
	MemdirRelevanceModeOverride string
	// MemdirAlreadySurfaced seeds paths excluded from repeated memdir selection in-session (H8).
	MemdirAlreadySurfaced map[string]struct{}
	// MemdirStrictLLM skips heuristic fallback after LLM memdir failure; also when RABBIT_CODE_MEMDIR_STRICT_LLM is set.
	MemdirStrictLLM bool
	// MemdirSelectMaxTokens caps the side-query completion (default 256, findRelevantMemories.ts max_tokens).
	MemdirSelectMaxTokens int
	// MemdirOnRecallShape optional (candidates, selected) after each memdir recall (H8 telemetry hook).
	MemdirOnRecallShape memdir.RecallShapeHook
	// MaxAssistantTurns if > 0 sets query.LoopState.MaxTurns for each Submit (caps assistant API rounds).
	MaxAssistantTurns int
	// TaskBudgetTotal if > 0 sets output_config.task_budget on main turn-loop Messages API calls (query.ts / QueryEngine.ts taskBudget).
	TaskBudgetTotal int
	// SkipCacheWrite remaps cache breakpoints per query.ts / claude.ts fork semantics before each assistant call.
	SkipCacheWrite bool
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
	// AfterSessionMemoryCompactSuccess optional notify/mark when SM compaction applies (H3 / autoCompact.ts).
	AfterSessionMemoryCompactSuccess AfterSessionMemoryCompactSuccess
	// PostCompactCleanup optional hook after any successful compact executor / session-memory compact (H3).
	PostCompactCleanup PostCompactCleanup
	// PostCompactCleanupHooks optional TS-ordered steps (postCompactCleanup.ts); runs before PostCompactCleanup when both set.
	PostCompactCleanupHooks *compact.PostCompactCleanupHooks
	// MicrocompactEditBuffer optional; reset on successful compact and wired to AnthropicAssistant when possible (H4).
	MicrocompactEditBuffer *compact.MicrocompactEditBuffer
	// InitialAutocompactConsecutiveFailures seeds the autocompact circuit when restoring a session (autoCompact.ts tracking).
	InitialAutocompactConsecutiveFailures int
	// RestoredAutoCompactTracking optional; cloned into each Submit's LoopState and used to seed consecutive failure count.
	RestoredAutoCompactTracking *compact.AutoCompactTracking
	// RestoredSnipRemovalLog optional; prepended into each Submit's LoopState.SnipRemovalLog for session continuity (H7).
	RestoredSnipRemovalLog []query.SnipRemovalEntry
	// RestoredSessionLastAssistantAt optional; seeds time-based microcompact wall-clock across process restarts (RFC3339 in JSON at app boundary).
	RestoredSessionLastAssistantAt time.Time
	// ExtractMemoriesSaved runs after a successful forked extract when new topic files were written (extractMemories appendSystemMessage analogue).
	ExtractMemoriesSaved func(memoryPaths []string, teamMemoryCount int)
	// CommandLifecycleNotify optional; invoked as notifyCommandLifecycle(uuid, phase) for each ConsumedCommandUUID in SubmitWithOptions
	// after EventKindDone (successful Submit), matching query.ts after query() returns normally.
	CommandLifecycleNotify func(uuid string, phase string)
	// ProcessUserInputHook optional; runs before memdir / template pipeline. When replace is true, newText is the submit body (QueryEngine.ts processUserInput analogue).
	ProcessUserInputHook ProcessUserInputHook
	// TruncateProcessUserInputHookOutput when true, applies processuserinput.TruncateHookOutput to replaced text from ProcessUserInputHook (UserPromptSubmit hook output cap).
	TruncateProcessUserInputHookOutput bool
	// ExtraTemplateNames returns extra template basenames (no .md) merged with features.TemplateNames() for TEMPLATES appendix and EventKindTemplatesActive (classifier hook surface).
	ExtraTemplateNames ExtraTemplateNames
	// AfterToolResultsHook runs after each tool round appends user tool_result blocks, before next-turn state reset (query.ts post-tools collect hooks).
	AfterToolResultsHook AfterToolResultsHook
}

// ProcessUserInputHook runs before memdir path resolution for each Submit (QueryEngine.ts processUserInput).
type ProcessUserInputHook func(ctx context.Context, userText string) (newText string, replace bool, err error)

// ExtraTemplateNames supplies template names keyed off resolved user text after memdir injection (before template appendix is applied).
type ExtraTemplateNames func(resolvedUserText string) []string

// AfterToolResultsHook observes the transcript immediately after tool results are merged (query.ts timing for skillPrefetch / taskSummary modules).
type AfterToolResultsHook func(ctx context.Context, st *query.LoopState, transcriptJSON json.RawMessage) error

// SubmitOptions optional per-submit fields (query.ts consumedCommandUuids + notifyCommandLifecycle).
type SubmitOptions struct {
	// ConsumedCommandUUIDs triggers CommandLifecycleNotify(uuid, "completed") for each non-empty id after successful Done.
	ConsumedCommandUUIDs []string
}

// Engine coordinates cancellable query turns (stub or real StreamAssistant / RunTurnLoop).
type Engine struct {
	ctx                              context.Context
	cancel                           context.CancelFunc
	ch                               chan EngineEvent
	wg                               sync.WaitGroup
	deps                             query.Deps
	model                            string
	maxTokens                        int
	stubDelay                        time.Duration
	memdirExplicitPaths              []string
	memdirMemoryDir                  string
	memdirProjectRoot                string // for memory system prompt "## Searching past context"
	memdirRecentTools                []string
	memdirTextComplete               memdir.TextCompleteFunc
	memdirRelevanceMode              memdir.RelevanceMode
	memdirSurfaced                   map[string]struct{}
	memdirStrictLLM                  bool
	memdirSelectMaxTokens            int
	memdirOnRecallShape              memdir.RecallShapeHook
	compactAdvisor                   func(query.LoopState, []byte) (bool, bool)
	compactExecutor                  CompactExecutor
	stopHooks                        []StopHookFunc
	recoverStrategy                  RecoverStrategy
	orphanPermissionAdvisor          func(query.LoopState) (string, bool)
	maxAssistantTurns                int
	taskBudgetTotal                  int
	skipCacheWrite                   bool
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
	afterSessionMemoryCompactSuccess AfterSessionMemoryCompactSuccess
	postCompactCleanup               PostCompactCleanup
	postCompactHooks                 *compact.PostCompactCleanupHooks
	microcompactEditBuffer           *compact.MicrocompactEditBuffer
	sessionLastAssistantAt           time.Time // wall clock of last model assistant (cross-Submit time-based MC)
	cacheBreakSeen                   int32     // atomic: prompt-cache break callback ran this Submit
	// autoCompactConsecutiveFailures counts failed proactive auto compact executor runs across Submits (H3 / autoCompact.ts);
	// mirrored onto st.AutoCompactTracking.ConsecutiveFailures when st != nil.
	autoCompactConsecutiveFailures int
	restoredAutoCompactTracking    *compact.AutoCompactTracking
	lastAutoCompactTracking        *compact.AutoCompactTracking // snapshot after last Submit for persistence
	lastSnipRemovalLog             []query.SnipRemovalEntry     // snapshot after last Submit (H7)
	persistSnapshotMu              sync.Mutex                   // last* fields: Done may be observed before defer runs (overlapping Submits)
	restoredSnipRemovalLog         []query.SnipRemovalEntry
	// streamOutputTotal accumulates UsageDelta.OutputTokens via chained Anthropic OnStreamUsage (H5.5).
	streamOutputTotal atomic.Int64

	extractCtl             *memdir.ExtractController
	initialSettings        map[string]interface{}
	extractMemoriesSavedFn func(memoryPaths []string, teamMemoryCount int)

	// Post-compact attachment state (compact.ts readFileState + plan + deltas); see post_compact_runtime.go.
	postCompactMu           sync.Mutex
	postCompactReads        map[string]postCompactReadEntry
	postCompactPlanPath     string
	postCompactPlanContent  string
	postCompactPlanMode     bool
	postCompactSkills       []compact.PostCompactSkillEntry
	postCompactDeltaAttach  []json.RawMessage
	postCompactWorkspaceDir string

	commandLifecycleNotify       func(uuid string, phase string)
	processUserInputHook         ProcessUserInputHook
	truncateProcessUserInputHook bool
	extraTemplateNames           ExtraTemplateNames
	afterToolResultsHook         AfterToolResultsHook
}

// NewEngine is equivalent to New(parent, nil) (stub assistant).
func NewEngine(parent context.Context) *Engine {
	return New(parent, nil)
}

// New constructs an engine. Nil cfg or nil cfg.Assistant uses timed stub text.
// When Assistant is *anthropic.AnthropicAssistant and Turn is nil, Turn is promoted to the same pointer so tool_use streams work.
func New(parent context.Context, cfg *Config) *Engine {
	ctx, cancel := context.WithCancel(parent)
	e := &Engine{
		ctx:                 ctx,
		cancel:              cancel,
		ch:                  make(chan EngineEvent, 64),
		model:               "claude-3-5-haiku-20241022",
		maxTokens:           1024,
		stubDelay:           50 * time.Millisecond,
		memdirSurfaced:      make(map[string]struct{}),
		memdirRelevanceMode: memdir.RelevanceModeHeuristic,
		extractCtl:          &memdir.ExtractController{},
		postCompactReads:    make(map[string]postCompactReadEntry),
	}
	if cfg != nil {
		deps := cfg.Deps
		if deps.Turn == nil {
			if aa, ok := deps.Assistant.(*anthropic.AnthropicAssistant); ok {
				deps.Turn = aa
			}
		}
		if deps.Tools == nil && (deps.Turn != nil || deps.Assistant != nil) {
			deps.Tools = query.NewDefaultToolRunner()
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
		e.memdirExplicitPaths = append([]string(nil), cfg.MemdirPaths...)
		e.memdirMemoryDir = resolveEngineMemdirMemoryDir(cfg)
		e.memdirProjectRoot = engineMemdirProjectRoot(cfg)
		if e.memdirMemoryDir != "" {
			_ = memdir.EnsureMemoryDirExists(e.memdirMemoryDir)
		}
		if e.memdirMemoryDir != "" && features.TeamMemoryEnabledFromMerged(cfg.InitialSettings) && e.deps.Tools != nil {
			e.deps.Tools = &memdir.TeamMemSecretGuardRunner{
				Inner:      e.deps.Tools,
				AutoMemDir: e.memdirMemoryDir,
				Enabled:    true,
			}
		}
		e.memdirRecentTools = append([]string(nil), cfg.MemdirRecentTools...)
		e.memdirTextComplete = cfg.MemdirTextComplete
		e.memdirRelevanceMode = effectiveMemdirRelevanceMode(cfg.MemdirRelevanceModeOverride)
		for k := range cfg.MemdirAlreadySurfaced {
			e.memdirSurfaced[k] = struct{}{}
		}
		e.memdirStrictLLM = cfg.MemdirStrictLLM || features.MemdirStrictLLM()
		if cfg.MemdirSelectMaxTokens > 0 {
			e.memdirSelectMaxTokens = cfg.MemdirSelectMaxTokens
		} else {
			e.memdirSelectMaxTokens = 256
		}
		e.memdirOnRecallShape = cfg.MemdirOnRecallShape
		e.compactAdvisor = cfg.CompactAdvisor
		e.compactExecutor = cfg.CompactExecutor
		e.stopHooks = append([]StopHookFunc(nil), cfg.StopHooks...)
		if cfg.StopHook != nil {
			e.stopHooks = append(e.stopHooks, cfg.StopHook)
		}
		e.initialSettings = cfg.InitialSettings
		e.recoverStrategy = cfg.RecoverStrategy
		e.orphanPermissionAdvisor = cfg.OrphanPermissionAdvisor
		if cfg.MaxAssistantTurns > 0 {
			e.maxAssistantTurns = cfg.MaxAssistantTurns
		}
		if cfg.TaskBudgetTotal > 0 {
			e.taskBudgetTotal = cfg.TaskBudgetTotal
		}
		e.skipCacheWrite = cfg.SkipCacheWrite
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
		e.afterSessionMemoryCompactSuccess = cfg.AfterSessionMemoryCompactSuccess
		if e.sessionMemoryCompact == nil && features.SessionMemoryCompactionEnabled() {
			if md := resolveEngineMemdirMemoryDir(cfg); md != "" {
				hooks := memdir.SessionMemoryCompactHooksForMemoryDir(md)
				if hooks.GetSessionMemoryContent != nil {
					fn := compact.NewSessionMemoryCompactExecutor(hooks)
					e.sessionMemoryCompact = func(ctx context.Context, agentID, model string, th int, transcript json.RawMessage) (json.RawMessage, bool, error) {
						rep, ok, err := fn(ctx, agentID, model, th, transcript)
						if len(rep) == 0 {
							return nil, ok, err
						}
						return json.RawMessage(rep), ok, err
					}
				}
			}
		}
		e.postCompactCleanup = cfg.PostCompactCleanup
		e.postCompactHooks = cfg.PostCompactCleanupHooks
		e.microcompactEditBuffer = cfg.MicrocompactEditBuffer
		if aa, ok := e.deps.Assistant.(*anthropic.AnthropicAssistant); ok && cfg.MicrocompactEditBuffer != nil {
			aa.MicrocompactBuffer = cfg.MicrocompactEditBuffer
		}
		if aa, ok := e.deps.Turn.(*anthropic.AnthropicAssistant); ok && cfg.MicrocompactEditBuffer != nil {
			aa.MicrocompactBuffer = cfg.MicrocompactEditBuffer
		}
		e.autoCompactConsecutiveFailures = cfg.InitialAutocompactConsecutiveFailures
		if t := cfg.RestoredAutoCompactTracking; t != nil && t.ConsecutiveFailures != nil {
			e.autoCompactConsecutiveFailures = *t.ConsecutiveFailures
		}
		e.restoredAutoCompactTracking = compact.CloneAutoCompactTracking(cfg.RestoredAutoCompactTracking)
		e.restoredSnipRemovalLog = query.CloneSnipRemovalLog(cfg.RestoredSnipRemovalLog)
		if !cfg.RestoredSessionLastAssistantAt.IsZero() {
			e.sessionLastAssistantAt = cfg.RestoredSessionLastAssistantAt
		}
		e.extractMemoriesSavedFn = cfg.ExtractMemoriesSaved
		e.commandLifecycleNotify = cfg.CommandLifecycleNotify
		e.processUserInputHook = cfg.ProcessUserInputHook
		e.truncateProcessUserInputHook = cfg.TruncateProcessUserInputHookOutput
		e.extraTemplateNames = cfg.ExtraTemplateNames
		e.afterToolResultsHook = cfg.AfterToolResultsHook
	}
	e.stopHooks = append(e.stopHooks, e.stopHookExtractMemories)
	return e
}

func (e *Engine) templateAppendixDir() string {
	if e.templateDir != "" {
		return e.templateDir
	}
	return features.TemplateMarkdownDir()
}

func (e *Engine) mergedTemplateNames(resolvedUserText string) []string {
	names := append([]string(nil), features.TemplateNames()...)
	if e.extraTemplateNames != nil {
		for _, n := range e.extraTemplateNames(resolvedUserText) {
			n = strings.TrimSpace(n)
			if n == "" {
				continue
			}
			names = append(names, n)
		}
	}
	seen := make(map[string]struct{})
	out := make([]string, 0, len(names))
	for _, n := range names {
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
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
	e.SubmitWithOptions(userText, SubmitOptions{})
}

// SubmitWithOptions is like Submit with per-submit options (e.g. ConsumedCommandUUIDs for command lifecycle parity).
func (e *Engine) SubmitWithOptions(userText string, opts SubmitOptions) {
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		modeTags := query.FormatHeadlessModeTags(query.UserTextHintFlags{
			ContextCollapse: features.ContextCollapseEnabled(),
			Ultrathink:      features.UltrathinkEnabled() || thinking.HasUltrathinkKeyword(userText),
			Ultraplan:       features.UltraplanEnabled(),
			SessionRestore:  features.SessionRestoreEnabled(),
		})
		if !e.trySend(EngineEvent{Kind: EventKindUserSubmit, UserText: userText, PhaseDetail: modeTags}) {
			return
		}
		if e.useQueryLoop() {
			e.runTurnLoop(userText, opts.ConsumedCommandUUIDs)
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
		if !e.trySend(EngineEvent{Kind: EventKindDone}) {
			return
		}
		e.fireCommandLifecycleNotifyCompleted(opts.ConsumedCommandUUIDs)
	}()
}

func (e *Engine) fireCommandLifecycleNotifyCompleted(uuids []string) {
	if e.commandLifecycleNotify == nil || len(uuids) == 0 {
		return
	}
	for _, u := range uuids {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		e.commandLifecycleNotify(u, "completed")
	}
}

func engineAutoMemoryEnabled(cfg *Config) bool {
	if cfg != nil && cfg.InitialSettings != nil {
		return features.AutoMemoryEnabledFromMerged(cfg.InitialSettings)
	}
	return features.AutoMemoryEnabled()
}

func engineMemdirProjectRoot(cfg *Config) string {
	if cfg == nil {
		return ""
	}
	if r := strings.TrimSpace(cfg.MemdirProjectRoot); r != "" {
		return filepath.Clean(r)
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return filepath.Clean(wd)
}

func resolveEngineMemdirMemoryDir(cfg *Config) string {
	if cfg == nil {
		return ""
	}
	if s := strings.TrimSpace(cfg.MemdirMemoryDir); s != "" {
		return s
	}
	if s := features.MemdirMemoryDirFromEnv(); s != "" {
		return s
	}
	if !engineAutoMemoryEnabled(cfg) {
		return ""
	}
	trusted := strings.TrimSpace(cfg.MemdirTrustedAutoMemoryDirectory)
	if !features.AutoMemdirFromProject() && trusted == "" {
		return ""
	}
	root := strings.TrimSpace(cfg.MemdirProjectRoot)
	if root == "" {
		wd, err := os.Getwd()
		if err != nil {
			return ""
		}
		root = wd
	}
	dir, err := memdir.ResolveAutoMemDirWithOptions(root, memdir.AutoMemResolveOptions{
		TrustedAutoMemoryDirectory: trusted,
	})
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(strings.TrimSpace(dir), string(filepath.Separator))
}

func effectiveMemdirRelevanceMode(override string) memdir.RelevanceMode {
	switch strings.ToLower(strings.TrimSpace(override)) {
	case "llm":
		return memdir.RelevanceModeLLM
	case "heuristic":
		return memdir.RelevanceModeHeuristic
	case "":
		if features.MemdirRelevanceMode() == "llm" {
			return memdir.RelevanceModeLLM
		}
		return memdir.RelevanceModeHeuristic
	default:
		return memdir.RelevanceModeHeuristic
	}
}

func (e *Engine) anthropicMemdirTextComplete() memdir.TextCompleteFunc {
	return func(ctx context.Context, systemPrompt, userMessage string) (string, error) {
		cl := e.anthropicClientPtr()
		if cl == nil {
			return "", fmt.Errorf("memdir: no anthropic client for LLM relevance")
		}
		combined := systemPrompt + "\n\n---\n\n" + userMessage
		msgs, err := json.Marshal([]map[string]any{
			{"role": "user", "content": []map[string]string{{"type": "text", "text": combined}}},
		})
		if err != nil {
			return "", err
		}
		mt := e.memdirSelectMaxTokens
		if mt <= 0 {
			mt = 256
		}
		body := anthropic.MessagesStreamBody{
			Model:     e.model,
			MaxTokens: mt,
			Messages:  msgs,
		}
		pol := e.anthropicPolicy()
		if pol.MaxAttempts == 0 {
			pol = anthropic.DefaultPolicy()
		}
		text, _, err := cl.PostMessagesStreamReadAssistant(ctx, body, pol)
		return text, err
	}
}

func (e *Engine) memdirPathsForSubmit(userText string) ([]string, error) {
	var parts [][]string
	if e.memdirMemoryDir != "" {
		mode := e.memdirRelevanceMode
		tc := e.memdirTextComplete
		if mode == memdir.RelevanceModeLLM && tc == nil {
			tc = e.anthropicMemdirTextComplete()
		}
		opts := memdir.FindRelevantMemoriesOpts{
			Mode:            mode,
			Limit:           5,
			RecentTools:     e.memdirRecentTools,
			AlreadySurfaced: e.memdirSurfaced,
			TextComplete:    tc,
			StrictLLM:       e.memdirStrictLLM,
			OnRecallShape:   e.memdirOnRecallShape,
		}
		rel, err := memdir.FindRelevantMemories(e.ctx, userText, e.memdirMemoryDir, opts)
		if err != nil {
			return nil, err
		}
		parts = append(parts, rel)
	}
	parts = append(parts, e.memdirExplicitPaths)
	var flat []string
	for _, p := range parts {
		flat = append(flat, p...)
	}
	return memdir.DedupePathsStable(flat), nil
}

func (e *Engine) applyMemdirWithPaths(userText string, paths []string) (resolved string, nFrag int, injectRawBytes int, err error) {
	if len(paths) == 0 {
		return userText, 0, 0, nil
	}
	frags, injectRawBytes, err := memdir.SessionFragmentsFromPathsWithAttachmentHeaders(paths)
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
		OnToolDone: func(name, id string, inputJSON, result []byte) {
			e.trySend(EngineEvent{
				Kind:           EventKindToolCallDone,
				ToolName:       name,
				ToolUseID:      id,
				ToolResultJSON: string(result),
			})
			e.recordPostCompactReadTool(name, inputJSON, result)
		},
		OnToolError: func(name, id string, err error) {
			e.trySend(EngineEvent{
				Kind:      EventKindToolCallFailed,
				ToolName:  name,
				ToolUseID: id,
				Err:       err,
			})
			if oid, ok := query.OrphanToolUseID(err); ok && oid != "" {
				e.trySend(EngineEvent{
					Kind:            EventKindOrphanPermission,
					OrphanToolUseID: oid,
				})
			}
		},
		OnHistorySnip: func(before, after, rounds int, snipID string) {
			e.trySend(EngineEvent{
				Kind:         EventKindHistorySnipApplied,
				PhaseDetail:  fmt.Sprintf("rounds=%d", rounds),
				PhaseAuxInt:  before,
				PhaseAuxInt2: after,
				SnipID:       snipID,
			})
		},
		OnSnipCompact: func(before, after, rounds int, snipID string) {
			e.trySend(EngineEvent{
				Kind:         EventKindSnipCompactApplied,
				PhaseDetail:  fmt.Sprintf("rounds=%d", rounds),
				PhaseAuxInt:  before,
				PhaseAuxInt2: after,
				SnipID:       snipID,
			})
		},
		OnPromptCacheBreakRecovery: func(phase string) {
			e.trySend(EngineEvent{Kind: EventKindPromptCacheBreakRecovery, PhaseDetail: phase})
		},
		OnAfterToolResults: func(ctx context.Context, st *query.LoopState, raw json.RawMessage) error {
			if e.afterToolResultsHook == nil {
				return nil
			}
			return e.afterToolResultsHook(ctx, st, raw)
		},
	}
}

func (e *Engine) anthropicClientPtr() *anthropic.Client {
	if a, ok := e.deps.Turn.(*anthropic.AnthropicAssistant); ok && a != nil && a.Client != nil {
		return a.Client
	}
	if a, ok := e.deps.Assistant.(*anthropic.AnthropicAssistant); ok && a != nil && a.Client != nil {
		return a.Client
	}
	return nil
}

func (e *Engine) anthropicPolicy() anthropic.Policy {
	if a, ok := e.deps.Turn.(*anthropic.AnthropicAssistant); ok && a != nil && a.Policy.MaxAttempts != 0 {
		return a.Policy
	}
	if a, ok := e.deps.Assistant.(*anthropic.AnthropicAssistant); ok && a != nil && a.Policy.MaxAttempts != 0 {
		return a.Policy
	}
	if a, ok := e.deps.Turn.(*anthropic.AnthropicAssistant); ok && a != nil {
		return a.Policy
	}
	if a, ok := e.deps.Assistant.(*anthropic.AnthropicAssistant); ok && a != nil {
		return a.Policy
	}
	return anthropic.Policy{}
}

// chainStreamUsage wraps Anthropic Client.OnStreamUsage to accumulate OutputTokens for H5.5 turn budget tracking.
func (e *Engine) chainStreamUsage() (restore func()) {
	c := e.anthropicClientPtr()
	if c == nil {
		return func() {}
	}
	prev := c.OnStreamUsage
	c.OnStreamUsage = func(u anthropic.UsageDelta) {
		e.streamOutputTotal.Add(u.OutputTokens)
		if prev != nil {
			prev(u)
		}
	}
	return func() { c.OnStreamUsage = prev }
}

func (e *Engine) submitTokenEstimate(ctx context.Context, mode, resolved string, injectRaw int) (total int, detail string) {
	detail = mode
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "api":
		cl := e.anthropicClientPtr()
		if cl != nil {
			msgsJSON, err := query.InitialUserMessagesJSON(resolved)
			if err == nil {
				pol := e.anthropicPolicy()
				if pol.MaxAttempts == 0 {
					pol = anthropic.DefaultPolicy()
				}
				n, err := cl.CountMessagesInputTokens(ctx, e.model, msgsJSON, pol)
				if err == nil {
					return n + query.EstimateAttachmentRawBytesAsTokens(injectRaw), "api"
				}
			}
		}
		fb := query.EstimateSubmitTokenBudgetTotal("bytes4", resolved, injectRaw)
		return fb, "api+fallback"
	default:
		return query.EstimateSubmitTokenBudgetTotal(mode, resolved, injectRaw), mode
	}
}

func (e *Engine) runTurnLoop(userText string, consumedCommandUUIDs []string) {
	restoreUsage := e.chainStreamUsage()
	defer restoreUsage()
	e.refreshMemorySystemPromptForAssistant()
	turnOutputBaseline := e.streamOutputTotal.Load()
	budgetTracker := query.NewBudgetTracker()

	st := &query.LoopState{}
	if e.restoredAutoCompactTracking != nil {
		st.AutoCompactTracking = compact.CloneAutoCompactTracking(e.restoredAutoCompactTracking)
	}
	query.MirrorAutocompactConsecutiveFailures(st, e.autoCompactConsecutiveFailures)
	var loopErr error
	defer func() {
		tr := compact.CloneAutoCompactTracking(st.AutoCompactTracking)
		sn := query.CloneSnipRemovalLog(st.SnipRemovalLog)
		e.persistSnapshotMu.Lock()
		e.lastAutoCompactTracking = tr
		e.lastSnipRemovalLog = sn
		e.persistSnapshotMu.Unlock()
		e.invokeStopHooks(st, loopErr)
	}()

	st.SnipRemovalLog = query.CloneSnipRemovalLog(e.restoredSnipRemovalLog)

	submitBody := userText
	if e.processUserInputHook != nil {
		repl, use, err := e.processUserInputHook(e.ctx, userText)
		if err != nil {
			loopErr = err
			e.trySend(EngineEvent{Kind: EventKindError, Err: err})
			return
		}
		if use {
			submitBody = repl
			if e.truncateProcessUserInputHook {
				submitBody = processuserinput.TruncateHookOutput(submitBody)
			}
		}
	}

	parsedOutputBudget, haveOutputBudget := query.ParseTokenBudget(submitBody)
	if !haveOutputBudget || parsedOutputBudget <= 0 {
		parsedOutputBudget = 0
		haveOutputBudget = false
	}

	paths, err := e.memdirPathsForSubmit(submitBody)
	if err != nil {
		loopErr = err
		e.trySend(EngineEvent{Kind: EventKindError, Err: err})
		return
	}
	resolved, nFrag, injectRaw, err := e.applyMemdirWithPaths(submitBody, paths)
	if err != nil {
		loopErr = err
		e.trySend(EngineEvent{Kind: EventKindError, Err: err})
		return
	}
	if nFrag > 0 {
		for _, p := range paths {
			if p != "" {
				e.memdirSurfaced[p] = struct{}{}
			}
		}
		if !e.trySend(EngineEvent{Kind: EventKindMemdirInject, MemdirFragmentCount: nFrag}) {
			return
		}
	}

	if dir := e.templateAppendixDir(); dir != "" {
		names := e.mergedTemplateNames(resolved)
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

	if features.TokenBudgetEnabled() {
		mode := features.SubmitTokenEstimateMode()
		totalTok, modeDetail := e.submitTokenEstimate(e.ctx, mode, resolved, injectRaw)
		e.trySend(EngineEvent{
			Kind:         EventKindSubmitTokenBudgetSnapshot,
			PhaseAuxInt:  totalTok,
			PhaseAuxInt2: injectRaw,
			PhaseDetail:  modeDetail,
		})
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
	if maxT := features.TokenBudgetMaxInputTokens(); maxT > 0 {
		mode := features.SubmitTokenEstimateMode()
		totalTok, _ := e.submitTokenEstimate(e.ctx, mode, resolved, injectRaw)
		if totalTok > maxT {
			loopErr = ErrTokenBudgetExceeded
			e.trySend(EngineEvent{Kind: EventKindError, Err: loopErr})
			return
		}
	}

	if features.BreakCacheCommandEnabled() {
		e.trySend(EngineEvent{Kind: EventKindBreakCacheCommand, PhaseDetail: "submit"})
	}
	if tplNames := e.mergedTemplateNames(resolved); len(tplNames) > 0 {
		e.trySend(EngineEvent{Kind: EventKindTemplatesActive, PhaseDetail: strings.Join(tplNames, ",")})
	}
	if features.CachedMicrocompactEnabled() {
		e.trySend(EngineEvent{Kind: EventKindCachedMicrocompactActive, PhaseDetail: anthropic.BetaCachedMicrocompactBody})
	}

	resolved = query.ApplyUserTextHints(resolved, query.UserTextHintFlags{
		ContextCollapse: features.ContextCollapseEnabled(),
		Ultrathink:      features.UltrathinkEnabled() || thinking.HasUltrathinkKeyword(resolved),
		Ultraplan:       features.UltraplanEnabled(),
		SessionRestore:  features.SessionRestoreEnabled(),
	})

	atomic.StoreInt32(&e.cacheBreakSeen, 0)
	ctxLoop := e.ctx
	if features.PromptCacheBreakDetectionEnabled() || features.PromptCacheBreakSuggestCompactEnabled() {
		ctxLoop = anthropic.ContextWithOnPromptCacheBreak(e.ctx, func() {
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
	var continuationSeed json.RawMessage
	for round := 0; round < maxSubmitContinuationRounds; round++ {
		var subErr error
		msgs, succeeded, subErr = e.executeRunTurnLoopAttempts(ctxLoop, st, resolved, continuationSeed, maxAttempts)
		continuationSeed = nil
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
			e.persistSessionLastAssistantAt(st)
			e.trySend(EngineEvent{Kind: EventKindDone, LoopTurnCount: st.TurnCount, PhaseDetail: stopReason})
			e.fireCommandLifecycleNotifyCompleted(consumedCommandUUIDs)
			return
		}
		needStopHookBlock := blockFromAfterTurn || (e.stopHookBlockingContinue != nil && e.stopHookBlockingContinue(e.ctx, *st))
		if needStopHookBlock {
			query.RecordLoopContinue(st, query.LoopContinue{Reason: query.ContinueReasonStopHookBlocking})
			PrepareLoopStateForStopHookBlockingContinuation(st)
			continue
		}

		if features.TokenBudgetEnabled() && haveOutputBudget && e.anthropicClientPtr() != nil && strings.TrimSpace(e.agentID) == "" {
			turnOut := int(e.streamOutputTotal.Load() - turnOutputBaseline)
			decision := query.CheckTokenBudget(&budgetTracker, e.agentID, parsedOutputBudget, turnOut)
			if decision.Action == query.BudgetActionContinue {
				next, aerr := query.AppendMetaUserTextMessage(msgs, decision.NudgeMessage)
				if aerr != nil {
					loopErr = aerr
					e.trySend(EngineEvent{Kind: EventKindError, Err: loopErr})
					return
				}
				continuationSeed = next
				query.RecordLoopContinue(st, query.LoopContinue{Reason: query.ContinueReasonTokenBudgetContinuation})
				PrepareLoopStateForTokenBudgetContinuation(st)
				e.trySend(EngineEvent{
					Kind:         EventKindTokenBudgetContinue,
					PhaseDetail:  decision.NudgeMessage,
					PhaseAuxInt:  decision.Pct,
					PhaseAuxInt2: decision.ContinuationCount,
				})
				continue
			}
			if decision.Completion != nil {
				c := decision.Completion
				e.trySend(EngineEvent{
					Kind: EventKindTokenBudgetCompleted,
					PhaseDetail: fmt.Sprintf("continuations=%d pct=%d turn=%d budget=%d diminishing=%t durMs=%d",
						c.ContinuationCount, c.Pct, c.TurnTokens, c.Budget, c.DiminishingReturns, c.DurationMs),
				})
			}
			break
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
			ctxCompact := compact.ContextWithExecutorSuggestMeta(e.ctx, compact.ExecutorSuggestMeta{
				AutoCompact:     false,
				ReactiveCompact: true,
			})
			sum, _, exErr := e.compactExecutor(ctxCompact, execPh, msgs)
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

	msgs = e.runCompactSuggestAfterSuccessfulTurn(st, msgs)

	e.persistSessionLastAssistantAt(st)
	e.trySend(EngineEvent{Kind: EventKindDone, LoopTurnCount: st.TurnCount})
	e.fireCommandLifecycleNotifyCompleted(consumedCommandUUIDs)
}

func (e *Engine) effectiveQuerySource(st *query.LoopState) string {
	if st != nil {
		if s := strings.TrimSpace(st.ToolUseContext.QuerySource); s != "" {
			return s
		}
	}
	return e.querySource
}

func (e *Engine) persistSessionLastAssistantAt(st *query.LoopState) {
	if st == nil || st.LastAssistantAt.IsZero() {
		return
	}
	e.sessionLastAssistantAt = st.LastAssistantAt
}

func (e *Engine) afterCompactSuccess(st *query.LoopState) {
	src := e.effectiveQuerySource(st)
	main := compact.IsMainThreadPostCompactSource(src)
	compact.RunPostCompactCleanup(e.ctx, src, e.microcompactEditBuffer, e.postCompactHooks)
	if e.postCompactCleanup != nil {
		e.postCompactCleanup(e.ctx, src, e.agentID, main)
	}
}

func (e *Engine) autoCompactCircuitTripped() bool {
	return e.autoCompactConsecutiveFailures >= compact.MaxConsecutiveAutocompactFailures
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

// AutoCompactTrackingForPersistence returns a deep copy of autocompact tracking after the last completed Submit
// (for session save). Nil if no Submit has finished.
func (e *Engine) AutoCompactTrackingForPersistence() *compact.AutoCompactTracking {
	e.persistSnapshotMu.Lock()
	p := e.lastAutoCompactTracking
	e.persistSnapshotMu.Unlock()
	return compact.CloneAutoCompactTracking(p)
}

// SnipRemovalLogForPersistence returns a deep copy of the snip removal log after the last completed Submit (H7 session sidecar).
func (e *Engine) SnipRemovalLogForPersistence() []query.SnipRemovalEntry {
	e.persistSnapshotMu.Lock()
	s := e.lastSnipRemovalLog
	e.persistSnapshotMu.Unlock()
	return query.CloneSnipRemovalLog(s)
}

// LastAssistantAtForPersistence returns the wall-clock time of the last model assistant message used for
// time-based microcompact (session carry-over). Populated from RestoredSessionLastAssistantAt at engine init
// and updated after each successful Submit that records an assistant turn. Zero means unknown / never set.
func (e *Engine) LastAssistantAtForPersistence() time.Time {
	return e.sessionLastAssistantAt
}

// Cancel stops in-flight Submit work (idempotent). In-flight HTTP streams should respect the same context when wired through RunTurnLoop.
func (e *Engine) Cancel() {
	e.cancel()
}

// Wait blocks until all Submit goroutines finish after Cancel.
func (e *Engine) Wait() {
	e.wg.Wait()
}
