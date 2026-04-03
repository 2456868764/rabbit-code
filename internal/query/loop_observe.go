package query

// LoopObservers receives optional callbacks for engine / TUI wiring (Phase 5).
type LoopObservers struct {
	OnAssistantText func(text string)
	OnToolStart     func(name, toolUseID string, input []byte)
	OnToolDone      func(name, toolUseID string, result []byte)
	OnToolError     func(name, toolUseID string, err error)
	// OnHistorySnip fires when P5.F.10 trimming removed leading messages (bytes UTF-8 length, rounds = drop count).
	OnHistorySnip func(bytesBefore, bytesAfter, rounds int)
	// OnSnipCompact fires when P5.2.2 SNIP_COMPACT trimming removed leading messages (independent env thresholds).
	OnSnipCompact func(bytesBefore, bytesAfter, rounds int)
	// OnPromptCacheBreakRecovery fires for H1 recovery steps: phase "trim_resend" or "compact_retry".
	OnPromptCacheBreakRecovery func(phase string)
}
