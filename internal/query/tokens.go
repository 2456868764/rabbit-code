package query

// EstimateUTF8BytesAsTokens is a coarse heuristic (~4 UTF-8 bytes per token for Latin-ish text).
// It is not a substitute for the API tokenizer; used for TOKEN_BUDGET and REACTIVE_COMPACT gates (continuation P5.F.1 / P5.F.2).
func EstimateUTF8BytesAsTokens(s string) int {
	if s == "" {
		return 0
	}
	tok := (len(s) + 3) / 4
	if tok < 1 {
		return 1
	}
	return tok
}

// EstimateTranscriptJSONTokens applies EstimateUTF8BytesAsTokens to the raw Messages JSON blob.
func EstimateTranscriptJSONTokens(transcriptJSON []byte) int {
	return EstimateUTF8BytesAsTokens(string(transcriptJSON))
}
