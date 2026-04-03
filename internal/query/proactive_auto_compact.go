package query

import "github.com/2456868764/rabbit-code/internal/features"

// ProactiveAutoCompactSuggested mirrors shouldAutoCompact headless inputs: transcript heuristic token count vs
// getAutoCompactThreshold (autoCompact.ts), gated like isAutoCompactEnabled + shouldAutoCompact suppressions.
// contextWindowTokens should be the capped context window (after RABBIT_CODE_AUTO_COMPACT_WINDOW); use 0 to resolve from model inside features (not recommended from engine — pass resolved window).
func ProactiveAutoCompactSuggested(transcriptJSON []byte, model string, maxOutputTokens int, contextWindowTokens int, snipTokensFreed int) bool {
	if !features.IsAutoCompactEnabled() {
		return false
	}
	if features.ContextCollapseEnabled() {
		return false
	}
	if features.SuppressProactiveAutoCompact() {
		return false
	}
	cw := contextWindowTokens
	if cw <= 0 {
		cw = features.ContextWindowTokensForModel(model)
	}
	cw = features.ApplyAutoCompactWindowCap(cw)
	effective := EffectiveContextInputWindow(cw, maxOutputTokens)
	threshold := AutoCompactThresholdTokens(effective, features.AutocompactPctOverride())
	if threshold <= 0 {
		return false
	}
	tok := EstimateTranscriptJSONTokens(transcriptJSON) - snipTokensFreed
	if tok < 0 {
		tok = 0
	}
	return tok >= threshold
}
