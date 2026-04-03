package query

// Constants aligned with services/compact/autoCompact.ts (headless subset).
const (
	AutocompactBufferTokens          = 13_000
	WarningThresholdBufferTokens     = 20_000
	ErrorThresholdBufferTokens       = 20_000
	ManualCompactBufferTokens        = 3_000
	maxOutputTokensForSummaryCap     = 20_000 // MAX_OUTPUT_TOKENS_FOR_SUMMARY in autoCompact.ts
)

// EffectiveContextInputWindow returns contextWindow minus reserved output headroom (capped), before autocompact buffer.
// Mirrors getEffectiveContextWindowSize’s subtraction of min(getMaxOutputTokensForModel(model), 20_000).
func EffectiveContextInputWindow(contextWindow, maxOutputForModel int) int {
	reserved := maxOutputForModel
	if reserved < 0 {
		reserved = 0
	}
	if reserved > maxOutputTokensForSummaryCap {
		reserved = maxOutputTokensForSummaryCap
	}
	out := contextWindow - reserved
	if out < 0 {
		return 0
	}
	return out
}

// AutoCompactThresholdTokens returns the token usage above which proactive autocompact should run
// (effectiveContextInputWindow − AUTOCOMPACT_BUFFER), with optional percentage cap like CLAUDE_AUTOCOMPACT_PCT_OVERRIDE.
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
// thresholdForPercent is getAutoCompactThreshold when auto is on, else effective input window (see TS).
// effectiveInputWindow is used for default blocking limit (effective − MANUAL_COMPACT_BUFFER).
// autoCompactThreshold is getAutoCompactThreshold(model); isAboveAutoCompact requires isAutoCompactEnabled && tokenUsage >= autoCompactThreshold.
// blockingLimitOverride > 0 replaces default blocking limit (CLAUDE_CODE_BLOCKING_LIMIT_OVERRIDE analogue).
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
	st.PercentLeft = (thresholdForPercent - tokenUsage) * 100 / thresholdForPercent
	if st.PercentLeft < 0 {
		st.PercentLeft = 0
	}
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
