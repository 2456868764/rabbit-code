package query

import (
	"github.com/2456868764/rabbit-code/internal/features"
)

// ToolTokenCountOverhead mirrors analyzeContext.ts TOOL_TOKEN_COUNT_OVERHEAD (API preamble when tools present).
const ToolTokenCountOverhead = 500

// HeadlessContextReport is a non-UI subset of analyzeContext ContextData: transcript metrics + threshold ladder.
type HeadlessContextReport struct {
	TranscriptBytes int
	EstimatedTokens int
	// StructuredMessageTokens is set when transcriptJSON parses as a messages array (microCompact.ts estimateMessageTokens path); 0 if unavailable.
	StructuredMessageTokens     int
	ContextWindowTokens         int
	EffectiveInputWindow        int
	AutoCompactThreshold        int
	ProactiveAutoCompactBlocked bool // querySource / reactive+cobalt / env gates
	TokenWarning                TokenWarningState
}

// BuildHeadlessContextReport mirrors analyzeContext-style totals for a transcript JSON blob (heuristic tokens only).
// tokenUsage should match the same basis as TS tokenCountWithEstimation when available; otherwise pass EstimatedTokens.
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
	r.EffectiveInputWindow = EffectiveContextInputWindow(cw, maxOutputTokens)
	r.AutoCompactThreshold = AutoCompactThresholdForProactive(model, maxOutputTokens, contextWindowTokens)
	r.ProactiveAutoCompactBlocked = !proactiveAutoCompactPreflight(querySource) ||
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
	r.TokenWarning = CalculateTokenWarningState(
		tokenUsage,
		thForPercent,
		r.EffectiveInputWindow,
		r.AutoCompactThreshold,
		features.IsAutoCompactEnabled(),
		features.BlockingLimitOverrideTokens(),
	)
	return r
}
