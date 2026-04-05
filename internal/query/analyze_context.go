package query

import (
	"fmt"
	"math"
	"strconv"
	"strings"

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
	return compact.AfterTurnReactiveCompactSuggested(transcriptJSON, minBytes, minTokens, false)
}

// TranscriptReactiveCompactSuggested is like ReactiveCompactByTranscript but respects LoopState.HasAttemptedReactiveCompact
// (query.ts hasAttemptedReactiveCompact: skip duplicate transcript-driven reactive compact in the same wave; H2).
func TranscriptReactiveCompactSuggested(st *LoopState, transcriptJSON []byte, minBytes, minTokens int) bool {
	hasAttempted := st != nil && st.HasAttemptedReactiveCompact
	return compact.AfterTurnReactiveCompactSuggested(transcriptJSON, minBytes, minTokens, hasAttempted)
}

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
	ProactiveAutoCompactBlocked bool
	TokenWarning                compact.TokenWarningState
}

// ResolvedTokenUsage is the token count used for threshold math (structured estimate when present, else bytes÷4 heuristic).
func (r HeadlessContextReport) ResolvedTokenUsage() int {
	if r.StructuredMessageTokens > 0 {
		return r.StructuredMessageTokens
	}
	return r.EstimatedTokens
}

// FormatHeadlessContextReportMarkdown mirrors context-noninteractive.ts formatContextAsMarkdownTable header + category table
// for the headless subset (no MCP/agents/skills rows — those need full analyzeContextUsage).
func FormatHeadlessContextReportMarkdown(model string, r HeadlessContextReport) string {
	m := strings.TrimSpace(model)
	if m == "" {
		m = "(default)"
	}
	usage := r.ResolvedTokenUsage()
	denom := r.EffectiveInputWindow
	if denom <= 0 {
		denom = 1
	}
	pct := int(math.Round(float64(usage) / float64(denom) * 100))
	if pct > 100 {
		pct = 100
	}

	var b strings.Builder
	b.WriteString("## Context Usage\n\n")
	b.WriteString("**Model:** ")
	b.WriteString(m)
	b.WriteString("  \n")
	fmt.Fprintf(&b, "**Tokens:** %s / %s (%d%%)\n", formatTokenCount(usage), formatTokenCount(r.EffectiveInputWindow), pct)
	fmt.Fprintf(&b, "**Context window (model cap):** %s\n", formatTokenCount(r.ContextWindowTokens))
	fmt.Fprintf(&b, "**Transcript bytes:** %d\n", r.TranscriptBytes)
	if r.AutoCompactThreshold > 0 {
		fmt.Fprintf(&b, "**Proactive autocompact threshold:** %s\n", formatTokenCount(r.AutoCompactThreshold))
	}
	if r.ProactiveAutoCompactBlocked {
		b.WriteString("**Proactive autocompact:** blocked (fork/source or feature gate)\n")
	} else {
		b.WriteString("**Proactive autocompact:** allowed\n")
	}
	tw := r.TokenWarning
	b.WriteString("\n### Token warning state\n\n")
	fmt.Fprintf(&b, "- percent left (vs threshold ladder): %d%%\n", tw.PercentLeft)
	fmt.Fprintf(&b, "- above warning: %v\n", tw.IsAboveWarningThreshold)
	fmt.Fprintf(&b, "- above error: %v\n", tw.IsAboveErrorThreshold)
	fmt.Fprintf(&b, "- above autocompact threshold: %v\n", tw.IsAboveAutoCompactThreshold)
	fmt.Fprintf(&b, "- at blocking limit: %v\n", tw.IsAtBlockingLimit)

	b.WriteString("\n### Estimated usage by category\n\n")
	b.WriteString("| Category | Tokens | Percentage |\n")
	b.WriteString("|----------|--------|------------|\n")
	free := r.EffectiveInputWindow - usage
	if free < 0 {
		free = 0
	}
	writeCat := func(name string, tok int) {
		p := float64(tok) / float64(denom) * 100
		fmt.Fprintf(&b, "| %s | %s | %.1f%% |\n", name, formatTokenCount(tok), p)
	}
	writeCat("Input (resolved estimate)", usage)
	if r.StructuredMessageTokens > 0 && r.StructuredMessageTokens != r.EstimatedTokens {
		writeCat("Heuristic (bytes÷4)", r.EstimatedTokens)
	}
	writeCat("Free space", free)
	return b.String()
}

func formatTokenCount(n int) string {
	if n < 0 {
		n = 0
	}
	return strconv.Itoa(n)
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
		features.ContextCollapseSuppressesProactiveAutocompact() ||
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
