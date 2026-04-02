package query

// LoopObservers receives optional callbacks for engine / TUI wiring (Phase 5).
type LoopObservers struct {
	OnAssistantText func(text string)
	OnToolStart     func(name, toolUseID string, input []byte)
	OnToolDone      func(name, toolUseID string, result []byte)
}
