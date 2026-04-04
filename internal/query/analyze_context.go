package query

import (
	"github.com/2456868764/rabbit-code/internal/features"
	"github.com/2456868764/rabbit-code/internal/services/compact"
)

// Re-export QuerySource constants for callers that set ToolUseContext.QuerySource (query.ts parity).
const (
	QuerySourceSessionMemory   = compact.QuerySourceSessionMemory
	QuerySourceCompact         = compact.QuerySourceCompact
	QuerySourceMarbleOrigami   = compact.QuerySourceMarbleOrigami
	QuerySourceExtractMemories = compact.QuerySourceExtractMemories
)

// ProactiveAutoCompactAllowedForQuerySource delegates to compact (autoCompact.ts).
func ProactiveAutoCompactAllowedForQuerySource(source string) bool {
	return compact.ProactiveAutoCompactAllowedForQuerySource(source)
}

// ReactiveCompactByTranscript mirrors a minimal analyzeContext-style gate: reactive compact is suggested
// when transcript JSON exceeds a byte threshold and/or a heuristic token threshold (continuation item 5).
func ReactiveCompactByTranscript(transcriptJSON []byte, minBytes, minTokens int) bool {
	if features.DisableCompact() {
		return false
	}
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

// ToolTokenCountOverhead mirrors analyzeContext.ts TOOL_TOKEN_COUNT_OVERHEAD (API preamble when tools present).
const ToolTokenCountOverhead = 500

// HeadlessContextReport is a non-UI subset of analyzeContext ContextData: transcript metrics + threshold ladder.
type HeadlessContextReport struct {
	TranscriptBytes int
	EstimatedTokens int
	// StructuredMessageTokens is set when transcriptJSON parses as a messages array (microCompact.ts estimateMessageTokens path); 0 if unavailable.
	StructuredMessageTokens int
	ContextWindowTokens     int
	EffectiveInputWindow    int
	AutoCompactThreshold    int
	ProactiveAutoCompactBlocked bool
	TokenWarning            compact.TokenWarningState
}

// BuildHeadlessContextReport mirrors analyzeContext-style totals for a transcript JSON blob (heuristic tokens only).
func BuildHeadlessContextReport(transcriptJSON []byte, model string, maxOutputTokens, contextWindowTokens int, tokenUsage int, querySource string) HeadlessContextReport {
	var r HeadlessContextReport
	r.TranscriptBytes = len(transcriptJSON)
	r.EstimatedTokens = EstimateTranscriptJSONTokens(transcriptJSON)
	if n, err := EstimateMessageTokensFromTranscriptJSON(transcriptJSON); err == nil {
		r.StructuredMessageTokens = n
	}
	if tokenUsage <= 0 {
		if r.StructuredMessageTokens > 0 {
			tokenUsage = r.StructuredMessageTokens
		} else {
			tokenUsage = r.EstimatedTokens
		}
	}
	cw := contextWindowTokens
	if cw <= 0 {
		cw = features.ContextWindowTokensForModel(model)
	}
	cw = features.ApplyAutoCompactWindowCap(cw)
	r.ContextWindowTokens = cw
	r.EffectiveInputWindow = compact.EffectiveContextInputWindow(cw, maxOutputTokens)
	r.AutoCompactThreshold = compact.AutoCompactThresholdForProactive(model, maxOutputTokens, contextWindowTokens)
	r.ProactiveAutoCompactBlocked = !compact.ProactiveAutoCompactPreflight(querySource) ||
		!features.IsAutoCompactEnabled() ||
		features.ContextCollapseEnabled() ||
		features.SuppressProactiveAutoCompact()

	thForPercent := r.EffectiveInputWindow
	if features.IsAutoCompactEnabled() {
		thForPercent = r.AutoCompactThreshold
	}
	if thForPercent <= 0 {
		thForPercent = r.EffectiveInputWindow
	}
	r.TokenWarning = compact.CalculateTokenWarningState(
		tokenUsage,
		thForPercent,
		r.EffectiveInputWindow,
		r.AutoCompactThreshold,
		features.IsAutoCompactEnabled(),
		features.BlockingLimitOverrideTokens(),
	)
	return r
}
