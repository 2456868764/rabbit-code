package engine

// EventKind classifies EngineEvent payloads for headless consumers (Phase 9 TUI will subscribe).
type EventKind int

const (
	EventKindUserSubmit EventKind = iota
	EventKindMemdirInject
	EventKindAssistantText
	EventKindToolCallStart
	EventKindToolCallDone
	EventKindCompactSuggest
	EventKindDone
	EventKindError
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
}
