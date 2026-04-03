package query

import "github.com/2456868764/rabbit-code/internal/features"

func proactiveAutoCompactPreflight(querySource string) bool {
	if !ProactiveAutoCompactAllowedForQuerySource(querySource) {
		return false
	}
	if features.ReactiveCompactEnabled() && features.TenguCobaltRaccoon() {
		return false
	}
	return true
}

// ProactiveAutoCompactSuggested mirrors shouldAutoCompact for the main loop (empty querySource).
func ProactiveAutoCompactSuggested(transcriptJSON []byte, model string, maxOutputTokens int, contextWindowTokens int, snipTokensFreed int) bool {
	return ProactiveAutoCompactSuggestedWithSource(transcriptJSON, model, maxOutputTokens, contextWindowTokens, snipTokensFreed, "")
}

// ProactiveAutoCompactSuggestedWithSource mirrors shouldAutoCompact including querySource fork gates and
// reactive-only cobalt suppression (autoCompact.ts); contextWindowTokens should already include AUTO_COMPACT_WINDOW cap.
func ProactiveAutoCompactSuggestedWithSource(transcriptJSON []byte, model string, maxOutputTokens int, contextWindowTokens int, snipTokensFreed int, querySource string) bool {
	if !proactiveAutoCompactPreflight(querySource) {
		return false
	}
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
