package query

import (
	"errors"
	"strings"

	"github.com/2456868764/rabbit-code/internal/features"
)

// ErrBlockingLimit is returned before the first assistant API call when transcript usage is at or past the
// manual-compact buffer limit (query.ts calculateTokenWarningState / isAtBlockingLimit synthetic PTL).
var ErrBlockingLimit = errors.New("query: blocking limit exceeded")

// BlockingLimitPreCheckApplies mirrors query.ts gates before the blocking-limit check (lines 628–635).
func BlockingLimitPreCheckApplies(querySource string, skipDueToPostCompactContinuation bool) bool {
	if skipDueToPostCompactContinuation {
		return false
	}
	qs := strings.TrimSpace(querySource)
	if qs == QuerySourceSessionMemory || qs == QuerySourceCompact || qs == QuerySourceExtractMemories {
		return false
	}
	// Reactive compact + auto on: let the real API / reactive path handle overflow (no synthetic preempt).
	if features.ReactiveCompactEnabled() && features.IsAutoCompactEnabled() {
		return false
	}
	// Context-collapse + auto on: real 413 / drain path must not be starved by synthetic preempt.
	if features.ContextCollapseEnabled() && features.IsAutoCompactEnabled() {
		return false
	}
	return true
}

// CheckBlockingLimitPreAssistant runs the numeric blocking ladder (auto_compact.ts calculateTokenWarningState).
// contextWindowTokens 0 means features.ContextWindowTokensForModel(model) + ApplyAutoCompactWindowCap.
func CheckBlockingLimitPreAssistant(
	model string,
	maxOutputTokens int,
	contextWindowTokens int,
	transcriptJSON []byte,
	snipTokensFreed int,
	querySource string,
	skipDueToPostCompactContinuation bool,
) error {
	if !BlockingLimitPreCheckApplies(querySource, skipDueToPostCompactContinuation) {
		return nil
	}
	tokenUsage := EstimateTranscriptJSONTokens(transcriptJSON) - snipTokensFreed
	if tokenUsage < 0 {
		tokenUsage = 0
	}
	if n, err := EstimateMessageTokensFromTranscriptJSON(transcriptJSON); err == nil && n > 0 {
		tokenUsage = n - snipTokensFreed
		if tokenUsage < 0 {
			tokenUsage = 0
		}
	}
	r := BuildHeadlessContextReport(transcriptJSON, model, maxOutputTokens, contextWindowTokens, tokenUsage, querySource)
	if r.TokenWarning.IsAtBlockingLimit {
		return ErrBlockingLimit
	}
	return nil
}
