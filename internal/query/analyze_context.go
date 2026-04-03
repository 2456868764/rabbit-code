package query

// ReactiveCompactByTranscript mirrors a minimal analyzeContext-style gate: reactive compact is suggested
// when transcript JSON exceeds a byte threshold and/or a heuristic token threshold (continuation item 5).
func ReactiveCompactByTranscript(transcriptJSON []byte, minBytes, minTokens int) bool {
	if minBytes > 0 && len(transcriptJSON) >= minBytes {
		return true
	}
	if minTokens > 0 && EstimateTranscriptJSONTokens(transcriptJSON) >= minTokens {
		return true
	}
	return false
}

// TranscriptReactiveCompactSuggested is like ReactiveCompactByTranscript but respects LoopState.HasAttemptedReactiveCompact
// (query.ts hasAttemptedReactiveCompact: skip duplicate transcript-driven reactive compact in the same wave; H2).
func TranscriptReactiveCompactSuggested(st *LoopState, transcriptJSON []byte, minBytes, minTokens int) bool {
	if st != nil && st.HasAttemptedReactiveCompact {
		return false
	}
	return ReactiveCompactByTranscript(transcriptJSON, minBytes, minTokens)
}
