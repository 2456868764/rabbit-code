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

// ParsePhase maps engine / event strings back to RunPhase (best-effort).
func ParsePhase(s string) RunPhase {
	switch s {
	case "idle", "":
		return RunIdle
	case "auto_pending":
		return RunAutoPending
	case "reactive_pending":
		return RunReactivePending
	case "executing":
		return RunExecuting
	default:
		return RunIdle
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

// AfterSuccessfulCompactExecution returns the phase after a successful executor run (H3 / services/compact-style cleanup).
// RunExecuting maps to RunIdle; other phases are unchanged.
func AfterSuccessfulCompactExecution(p RunPhase) RunPhase {
	if p == RunExecuting {
		return RunIdle
	}
	return p
}

// ExecutorPhaseAfterSchedule is the phase passed to CompactExecutor (pending → executing; H3).
func ExecutorPhaseAfterSchedule(scheduled RunPhase) RunPhase {
	switch scheduled {
	case RunAutoPending, RunReactivePending:
		return RunExecuting
	default:
		return scheduled
	}
}

// ResultPhaseAfterCompactExecutor is the phase for EventKindCompactResult: idle on success, else exec phase.
func ResultPhaseAfterCompactExecutor(execPhase RunPhase, execErr error) RunPhase {
	if execErr != nil {
		return execPhase
	}
	return AfterSuccessfulCompactExecution(execPhase)
}
