package compact

// RunPhase tracks compact scheduling state (services/compact/* Phase 5 skeleton).
type RunPhase int

const (
	RunIdle RunPhase = iota
	RunAutoPending
	RunReactivePending
	RunExecuting
)

func (p RunPhase) String() string {
	switch p {
	case RunIdle:
		return "idle"
	case RunAutoPending:
		return "auto_pending"
	case RunReactivePending:
		return "reactive_pending"
	case RunExecuting:
		return "executing"
	default:
		return "unknown"
	}
}

// Next returns the following phase for a successful scheduling edge (stub state machine).
func (p RunPhase) Next(auto, reactive bool) RunPhase {
	switch p {
	case RunIdle:
		if reactive {
			return RunReactivePending
		}
		if auto {
			return RunAutoPending
		}
		return RunIdle
	case RunAutoPending, RunReactivePending:
		return RunExecuting
	case RunExecuting:
		return RunIdle
	default:
		return RunIdle
	}
}
