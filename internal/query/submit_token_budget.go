package query

import "strings"

// EstimateAttachmentRawBytesAsTokens maps raw memdir / inject bytes to heuristic tokens (aligns with coarse 4 bytes/token; H5 / P5.F.1).
func EstimateAttachmentRawBytesAsTokens(rawBytes int) int {
	if rawBytes <= 0 {
		return 0
	}
	return (rawBytes + 3) / 4
}

// EstimateResolvedSubmitTextTokens selects token basis for resolved Submit text when TOKEN_BUDGET max-input-tokens is enforced.
// mode "structured" uses EstimateMessageTokensFromTranscriptJSON when resolved looks like a Messages JSON array; otherwise bytes4 heuristic.
func EstimateResolvedSubmitTextTokens(mode, resolved string) int {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "structured":
		s := strings.TrimSpace(resolved)
		if strings.HasPrefix(s, "[") {
			if n, err := EstimateMessageTokensFromTranscriptJSON([]byte(resolved)); err == nil && n > 0 {
				return n
			}
		}
	}
	return EstimateUTF8BytesAsTokens(resolved)
}

// EstimateSubmitTokenBudgetTotal is resolved-text estimate plus attachment pseudo-tokens (H5: attachments count toward same token cap when set).
// Mode "api" is not counted here; the engine uses Anthropic count_tokens and falls back to bytes4+inject.
func EstimateSubmitTokenBudgetTotal(mode, resolved string, injectRawBytes int) int {
	if strings.ToLower(strings.TrimSpace(mode)) == "api" {
		return EstimateUTF8BytesAsTokens(resolved) + EstimateAttachmentRawBytesAsTokens(injectRawBytes)
	}
	return EstimateResolvedSubmitTextTokens(mode, resolved) + EstimateAttachmentRawBytesAsTokens(injectRawBytes)
}
