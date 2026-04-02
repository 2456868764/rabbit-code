package query

// LoopObservers receives optional callbacks for engine / TUI wiring (Phase 5).
type LoopObservers struct {
	OnAssistantText func(text string)
	OnToolStart     func(name, toolUseID string, input []byte)
	OnToolDone      func(name, toolUseID string, result []byte)
	OnToolError     func(name, toolUseID string, err error)
	// OnHistorySnip fires when P5.F.10 trimming removed leading messages (bytes UTF-8 length, rounds = drop count).
	OnHistorySnip func(bytesBefore, bytesAfter, rounds int)
}
