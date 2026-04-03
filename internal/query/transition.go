package query

import "fmt"

// Transition names logical query loop edges (table-driven tests, AC5-1).
type Transition string

const (
	TranReceiveAssistant Transition = "receive_assistant"
	TranScheduleTools    Transition = "schedule_tools"
	TranToolCallsDone    Transition = "tool_calls_done"
	TranStartCompact     Transition = "start_compact"
	TranFinishCompact    Transition = "finish_compact"
)

// ApplyTransition returns the next state (pure, no I/O).
func ApplyTransition(s LoopState, t Transition) LoopState {
	out := s
	switch t {
	case TranReceiveAssistant:
		out.TurnCount++
	case TranScheduleTools:
		out.PendingTools++
	case TranToolCallsDone:
		out.PendingTools--
		if out.PendingTools < 0 {
			out.PendingTools = 0
		}
	case TranStartCompact:
		out.InCompact = true
		out.CompactCount++
		// AutoCompactTracking mirrors autoCompact.ts bookkeeping on compact start (H6).
		out.AutoCompactTracking = &AutoCompactTracking{
			Compacted:   true,
			TurnCounter: out.TurnCount,
			TurnID:      fmt.Sprintf("autocompact:%d", out.CompactCount),
		}
	case TranFinishCompact:
		out.InCompact = false
	default:
		// unknown: no-op
	}
	return out
}
