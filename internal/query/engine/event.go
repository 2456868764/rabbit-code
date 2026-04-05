package engine

// EventKind classifies EngineEvent payloads for headless consumers (Phase 9 TUI will subscribe).
type EventKind int

const (
	EventKindUserSubmit EventKind = iota
	EventKindMemdirInject
	EventKindAssistantText
	EventKindToolCallStart
	EventKindToolCallDone
	EventKindToolCallFailed
	EventKindOrphanPermission
	EventKindCompactSuggest
	EventKindCompactResult
	EventKindDone
	EventKindError
	// P5.F.6–F.10 headless signals (TUI / telemetry).
	EventKindBreakCacheCommand
	EventKindPromptCacheBreakDetected
	EventKindPromptCacheBreakRecovery
	EventKindTemplatesActive
	EventKindCachedMicrocompactActive
	EventKindHistorySnipApplied
	EventKindSnipCompactApplied
	// EventKindSubmitTokenBudgetSnapshot reports combined heuristic submit tokens (H5): PhaseAuxInt=total, PhaseAuxInt2=inject raw bytes, PhaseDetail=mode.
	EventKindSubmitTokenBudgetSnapshot
	// EventKindTokenBudgetContinue reports an automatic token-budget nudge continuation (H5.5); PhaseDetail=nudge, PhaseAuxInt=pct, PhaseAuxInt2=continuationCount.
	EventKindTokenBudgetContinue
	// EventKindTokenBudgetCompleted reports stop after budget continuations or diminishing returns (H5.5); PhaseDetail holds key=value telemetry.
	EventKindTokenBudgetCompleted
	// EventKindWebSearchProgress reports inner web search stream progress; ToolUseID=outer tool_use id, PhaseDetail=json of websearchtool.WebSearchProgress.
	EventKindWebSearchProgress
)

// EngineEvent is a single unit on the engine event channel.
type EngineEvent struct {
	Kind       EventKind
	UserText   string
	AssistText string
	Err        error `json:"-"`

	ToolName            string
	ToolUseID           string
	ToolInputJSON       string
	ToolResultJSON      string
	MemdirFragmentCount int

	CompactPhase           string
	SuggestAutoCompact     bool
	SuggestReactiveCompact bool

	// APIErrorKind is set for EventKindError when the failure unwraps to anthropic.APIError (P5.1.3).
	APIErrorKind string
	// RecoverableCompact hints prompt_too_long / max_output_tokens style recovery (compact / trim); TUI may react.
	RecoverableCompact bool
	// OrphanToolUseID is set for EventKindOrphanPermission (P5.3.3 stub).
	OrphanToolUseID string
	// LoopTurnCount is set on EventKindDone (assistant rounds finished in this Submit).
	LoopTurnCount int
	// CompactSummary is set on EventKindCompactResult when CompactExecutor runs (P5.2.1).
	CompactSummary string
	// PhaseDetail carries free-form text for P5.F.* signal kinds (e.g. template names, snip stats).
	PhaseDetail string
	// PhaseAuxInt / PhaseAuxInt2 optional integers (e.g. history snip before/after bytes).
	PhaseAuxInt  int
	PhaseAuxInt2 int
	// SnipID is set for EventKindHistorySnipApplied / EventKindSnipCompactApplied (H7 stable id).
	SnipID string
}
