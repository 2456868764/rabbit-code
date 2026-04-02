package compact

import "context"

// ExecuteStub is a Phase 5 executor that ignores transcript and returns a fixed summary (tests / wiring closure).
func ExecuteStub(_ context.Context, phase RunPhase, transcriptJSON []byte) (summary string, err error) {
	_ = phase
	_ = transcriptJSON
	return "[stub compact summary]", nil
}
