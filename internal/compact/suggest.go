package compact

import (
	"context"
	"fmt"

	"github.com/2456868764/rabbit-code/internal/query"
)

// ReactiveSuggestFromTranscript delegates to query.ReactiveCompactByTranscript for services/compact-style gating (item 14).
func ReactiveSuggestFromTranscript(transcript []byte, minBytes, minTokens int) bool {
	return query.ReactiveCompactByTranscript(transcript, minBytes, minTokens)
}

// FormatStubCompactSummary builds a deterministic summary string including transcript heuristics (tests / logging).
func FormatStubCompactSummary(phase RunPhase, transcript []byte) string {
	return fmt.Sprintf("[stub compact phase=%s bytes=%d estTok=%d]", phase.String(), len(transcript), query.EstimateTranscriptJSONTokens(transcript))
}

// ExecuteStubWithMeta is like ExecuteStub but embeds phase and transcript metrics in the summary.
func ExecuteStubWithMeta(_ context.Context, phase RunPhase, transcriptJSON []byte) (summary string, err error) {
	return FormatStubCompactSummary(phase, transcriptJSON), nil
}
