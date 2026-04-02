package query

// Transition names logical query loop edges (table-driven tests, AC5-1 seed).
type Transition string

const (
	TranReceiveAssistant Transition = "receive_assistant"
	TranScheduleTools    Transition = "schedule_tools"
	TranToolCallsDone    Transition = "tool_calls_done"
)

// ApplyTransition returns the next state (pure, no I/O).
func ApplyTransition(s LoopState, t Transition) LoopState {
	switch t {
	case TranReceiveAssistant:
		return LoopState{TurnCount: s.TurnCount + 1, PendingTools: s.PendingTools}
	case TranScheduleTools:
		return LoopState{TurnCount: s.TurnCount, PendingTools: s.PendingTools + 1}
	case TranToolCallsDone:
		p := s.PendingTools - 1
		if p < 0 {
			p = 0
		}
		return LoopState{TurnCount: s.TurnCount, PendingTools: p}
	default:
		return s
	}
}
