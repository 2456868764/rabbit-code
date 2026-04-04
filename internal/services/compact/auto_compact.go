package compact

import (
	"encoding/json"
	"math"
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
)

// Constants aligned with services/compact/autoCompact.ts (headless subset).
const (
	AutocompactBufferTokens           = 13_000
	WarningThresholdBufferTokens      = 20_000
	ErrorThresholdBufferTokens        = 20_000
	ManualCompactBufferTokens         = 3_000
	MaxOutputTokensForSummaryCap      = 20_000 // MAX_OUTPUT_TOKENS_FOR_SUMMARY in autoCompact.ts
	MaxConsecutiveAutocompactFailures = 3      // autoCompact.ts MAX_CONSECUTIVE_AUTOCOMPACT_FAILURES
)

// Query source strings mirror query.ts QuerySource where they gate shouldAutoCompact (autoCompact.ts).
const (
	QuerySourceSessionMemory   = "session_memory"
	QuerySourceCompact         = "compact"
	QuerySourceMarbleOrigami   = "marble_origami"
	QuerySourceExtractMemories = "extract_memories"
)

// estimateTranscriptJSONTokens matches query.EstimateTranscriptJSONTokens (bytes/4 heuristic) without importing query (avoids cycles).
func estimateTranscriptJSONTokens(transcriptJSON []byte) int {
	s := string(transcriptJSON)
	if s == "" {
		return 0
	}
	tok := (len(s) + 3) / 4
	if tok < 1 {
		return 1
	}
	return tok
}

// EffectiveContextInputWindow returns contextWindow minus reserved output headroom (capped), before autocompact buffer.
func EffectiveContextInputWindow(contextWindow, maxOutputForModel int) int {
	reserved := maxOutputForModel
	if reserved < 0 {
		reserved = 0
	}
	if reserved > MaxOutputTokensForSummaryCap {
		reserved = MaxOutputTokensForSummaryCap
	}
	out := contextWindow - reserved
	if out < 0 {
		return 0
	}
	return out
}

// AutoCompactThresholdTokens returns the token usage above which proactive autocompact should run.
func AutoCompactThresholdTokens(effectiveContextInputWindow int, pctOverride float64) int {
	if effectiveContextInputWindow <= 0 {
		return 0
	}
	autocompactThreshold := effectiveContextInputWindow - AutocompactBufferTokens
	if autocompactThreshold <= 0 {
		return 0
	}
	if pctOverride > 0 && pctOverride <= 100 {
		p := int(float64(effectiveContextInputWindow) * (pctOverride / 100))
		if p > 0 && p < autocompactThreshold {
			return p
		}
	}
	return autocompactThreshold
}

// TokenWarningState mirrors calculateTokenWarningState from autoCompact.ts (numeric core only).
type TokenWarningState struct {
	PercentLeft                 int
	IsAboveWarningThreshold     bool
	IsAboveErrorThreshold       bool
	IsAboveAutoCompactThreshold bool
	IsAtBlockingLimit           bool
}

// CalculateTokenWarningState mirrors autoCompact.ts calculateTokenWarningState for a pre-resolved threshold ladder.
func CalculateTokenWarningState(
	tokenUsage int,
	thresholdForPercent int,
	effectiveInputWindow int,
	autoCompactThreshold int,
	isAutoCompactEnabled bool,
	blockingLimitOverride int,
) TokenWarningState {
	var st TokenWarningState
	if thresholdForPercent <= 0 {
		return st
	}
	// Match autoCompact.ts: Math.max(0, Math.round(((threshold - tokenUsage) / threshold) * 100))
	st.PercentLeft = int(math.Max(0, math.Round(float64(thresholdForPercent-tokenUsage)/float64(thresholdForPercent)*100)))
	warningTh := thresholdForPercent - WarningThresholdBufferTokens
	errorTh := thresholdForPercent - ErrorThresholdBufferTokens
	st.IsAboveWarningThreshold = tokenUsage >= warningTh
	st.IsAboveErrorThreshold = tokenUsage >= errorTh
	st.IsAboveAutoCompactThreshold = isAutoCompactEnabled && tokenUsage >= autoCompactThreshold
	defaultBlocking := effectiveInputWindow - ManualCompactBufferTokens
	if defaultBlocking < 0 {
		defaultBlocking = 0
	}
	blocking := defaultBlocking
	if blockingLimitOverride > 0 {
		blocking = blockingLimitOverride
	}
	st.IsAtBlockingLimit = tokenUsage >= blocking
	return st
}

// AutoCompactTracking mirrors services/compact/autoCompact.ts AutoCompactTrackingState (headless bookkeeping, H6).
type AutoCompactTracking struct {
	Compacted           bool
	TurnCounter         int
	TurnID              string
	ConsecutiveFailures *int
}

// CloneAutoCompactTracking returns a deep copy (ConsecutiveFailures pointer duplicated).
func CloneAutoCompactTracking(p *AutoCompactTracking) *AutoCompactTracking {
	if p == nil {
		return nil
	}
	c := *p
	if p.ConsecutiveFailures != nil {
		v := *p.ConsecutiveFailures
		c.ConsecutiveFailures = &v
	}
	return &c
}

type autoCompactTrackingJSON struct {
	Compacted           bool   `json:"compacted,omitempty"`
	TurnCounter         int    `json:"turnCounter,omitempty"`
	TurnID              string `json:"turnId,omitempty"`
	ConsecutiveFailures *int   `json:"consecutiveFailures,omitempty"`
}

// MarshalAutoCompactTrackingJSON encodes tracking for session persistence (restore after restart).
func MarshalAutoCompactTrackingJSON(t *AutoCompactTracking) ([]byte, error) {
	if t == nil {
		return []byte("{}"), nil
	}
	j := autoCompactTrackingJSON{
		Compacted:   t.Compacted,
		TurnCounter: t.TurnCounter,
		TurnID:      t.TurnID,
	}
	if t.ConsecutiveFailures != nil {
		v := *t.ConsecutiveFailures
		j.ConsecutiveFailures = &v
	}
	return json.Marshal(j)
}

// UnmarshalAutoCompactTrackingJSON decodes session JSON into AutoCompactTracking (nil input/empty → nil).
func UnmarshalAutoCompactTrackingJSON(data []byte) (*AutoCompactTracking, error) {
	if len(data) == 0 || strings.TrimSpace(string(data)) == "" {
		return nil, nil
	}
	var j autoCompactTrackingJSON
	if err := json.Unmarshal(data, &j); err != nil {
		return nil, err
	}
	out := &AutoCompactTracking{
		Compacted:   j.Compacted,
		TurnCounter: j.TurnCounter,
		TurnID:      j.TurnID,
	}
	if j.ConsecutiveFailures != nil {
		v := *j.ConsecutiveFailures
		out.ConsecutiveFailures = &v
	}
	return out, nil
}

// RecompactionMeta mirrors autoCompact.ts RecompactionInfo fields used before compactConversation (headless subset).
type RecompactionMeta struct {
	IsRecompactionInChain     bool
	TurnsSincePreviousCompact int
	PreviousCompactTurnID     string
	AutoCompactThreshold      int
	QuerySource               string
}

// ProactiveAutoCompactAllowedForQuerySource is false for forked agents that would deadlock or corrupt
// shared state when proactive autocompact runs (session_memory, compact), and for marble_origami when
// CONTEXT_COLLAPSE is enabled (autoCompact.ts).
func ProactiveAutoCompactAllowedForQuerySource(source string) bool {
	switch source {
	case QuerySourceSessionMemory, QuerySourceCompact, QuerySourceExtractMemories:
		return false
	}
	if source == QuerySourceMarbleOrigami && features.ContextCollapseEnabled() {
		return false
	}
	return true
}

// ProactiveAutoCompactPreflight mirrors shouldAutoCompact pre-checks before threshold math (reactive+cobalt + fork gates).
func ProactiveAutoCompactPreflight(querySource string) bool {
	if !ProactiveAutoCompactAllowedForQuerySource(querySource) {
		return false
	}
	if features.ReactiveCompactEnabled() && features.TenguCobaltRaccoon() {
		return false
	}
	return true
}

// ProactiveAutoCompactSuggested mirrors shouldAutoCompact for the main loop (empty querySource).
// Token count uses the raw JSON byte heuristic only; for Messages-array parity with tokenCountWithEstimation, use
// ProactiveAutocompactFromUsage with query.EstimateMessageTokensFromTranscriptJSON (engine path).
func ProactiveAutoCompactSuggested(transcriptJSON []byte, model string, maxOutputTokens int, contextWindowTokens int, snipTokensFreed int) bool {
	return ProactiveAutoCompactSuggestedWithSource(transcriptJSON, model, maxOutputTokens, contextWindowTokens, snipTokensFreed, "")
}

// AfterTurnProactiveAutocompactFromUsage is the proactive usage gate for post-turn scheduling (autoCompact.ts):
// same as ProactiveAutocompactFromUsage but suppresses proactive autocompact when the session circuit has tripped.
func AfterTurnProactiveAutocompactFromUsage(tokenUsage int, model string, maxOutputTokens, contextWindowTokens int, querySource string, circuitTripped bool) bool {
	if circuitTripped {
		return false
	}
	return ProactiveAutocompactFromUsage(tokenUsage, model, maxOutputTokens, contextWindowTokens, querySource)
}

// AfterTurnReactiveCompactSuggested mirrors transcript-driven reactive compact after a successful turn without importing query.
// Caller passes whether this wave already attempted reactive compact (LoopState.HasAttemptedReactiveCompact).
func AfterTurnReactiveCompactSuggested(transcriptJSON []byte, minBytes, minTokens int, hasAttemptedReactive bool) bool {
	if hasAttemptedReactive {
		return false
	}
	if features.DisableCompact() {
		return false
	}
	if minBytes > 0 && len(transcriptJSON) >= minBytes {
		return true
	}
	if minTokens > 0 {
		tok := estimateTranscriptJSONTokens(transcriptJSON)
		if tok >= minTokens {
			return true
		}
	}
	return false
}

// ProactiveAutocompactFromUsage is the threshold half of shouldAutoCompact after preflight (autoCompact.ts): tokenUsage
// should already subtract snipTokensFreed. Callers typically derive usage from structured Messages JSON when valid.
func ProactiveAutocompactFromUsage(tokenUsage int, model string, maxOutputTokens, contextWindowTokens int, querySource string) bool {
	if !ProactiveAutoCompactPreflight(querySource) {
		return false
	}
	if !features.IsAutoCompactEnabled() {
		return false
	}
	if features.ContextCollapseSuppressesProactiveAutocompact() {
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
	tok := tokenUsage
	if tok < 0 {
		tok = 0
	}
	// autoCompact.ts: isAboveAutoCompactThreshold = isAutoCompactEnabled() && tokenUsage >= autoCompactThreshold
	return tok >= threshold
}

// ProactiveAutoCompactSuggestedWithSource mirrors shouldAutoCompact including querySource fork gates (autoCompact.ts).
// Uses byte/4 on the raw JSON blob; prefer ProactiveAutocompactFromUsage when the transcript is valid API messages JSON.
func ProactiveAutoCompactSuggestedWithSource(transcriptJSON []byte, model string, maxOutputTokens int, contextWindowTokens int, snipTokensFreed int, querySource string) bool {
	tok := estimateTranscriptJSONTokens(transcriptJSON) - snipTokensFreed
	if tok < 0 {
		tok = 0
	}
	return ProactiveAutocompactFromUsage(tok, model, maxOutputTokens, contextWindowTokens, querySource)
}

// AutoCompactThresholdForProactive returns getAutoCompactThreshold(model) headless inputs (autoCompact.ts).
func AutoCompactThresholdForProactive(model string, maxOutputTokens, contextWindowTokens int) int {
	cw := contextWindowTokens
	if cw <= 0 {
		cw = features.ContextWindowTokensForModel(model)
	}
	cw = features.ApplyAutoCompactWindowCap(cw)
	effective := EffectiveContextInputWindow(cw, maxOutputTokens)
	return AutoCompactThresholdTokens(effective, features.AutocompactPctOverride())
}
