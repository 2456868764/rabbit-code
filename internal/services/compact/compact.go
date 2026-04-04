package compact

import (
	"context"
	"fmt"
)

// RunPhase tracks compact scheduling state (services/compact/compact.ts scheduling subset, Phase 5).
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

// AfterSuccessfulCompactExecution returns the phase after a successful executor run (H3).
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

// ExecuteStub is a Phase 5 executor that ignores transcript and returns a fixed summary (tests / wiring closure).
func ExecuteStub(_ context.Context, phase RunPhase, transcriptJSON []byte) (summary string, nextTranscriptJSON []byte, err error) {
	_ = phase
	_ = transcriptJSON
	return "[stub compact summary]", nil, nil
}

// FormatStubCompactSummary builds a deterministic summary string including transcript heuristics (tests / logging).
func FormatStubCompactSummary(phase RunPhase, transcript []byte) string {
	return fmt.Sprintf("[stub compact phase=%s bytes=%d estTok=%d]", phase.String(), len(transcript), estimateTranscriptJSONTokens(transcript))
}

// ExecuteStubWithMeta is like ExecuteStub but embeds phase and transcript metrics in the summary.
func ExecuteStubWithMeta(_ context.Context, phase RunPhase, transcriptJSON []byte) (summary string, nextTranscriptJSON []byte, err error) {
	return FormatStubCompactSummary(phase, transcriptJSON), nil, nil
}
