package engine

// EventKind classifies EngineEvent payloads for headless consumers (Phase 9 TUI will subscribe).
type EventKind int

const (
	EventKindUserSubmit EventKind = iota
	EventKindAssistantText
	EventKindDone
)

// EngineEvent is a single unit on the engine event channel.
type EngineEvent struct {
	Kind       EventKind
	UserText   string
	AssistText string
}
