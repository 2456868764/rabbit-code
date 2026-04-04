package query

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/2456868764/rabbit-code/internal/services/compact"
)

// Heuristics: src/query/tokenBudget.ts (CheckTokenBudget), utils/tokenBudget.ts (parse / continuation message),
// microCompact.ts (estimateMessageTokens), submit path (H5).

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

// ImageDocumentTokenEstimate mirrors microCompact.ts IMAGE_MAX_TOKEN_SIZE (delegates to compact for single source).
const ImageDocumentTokenEstimate = compact.ImageDocumentTokenEstimate

// EstimateMessageTokensFromTranscriptJSON delegates to compact.EstimateMessageTokensFromAPIMessagesJSON (microCompact.ts estimateMessageTokens).
func EstimateMessageTokensFromTranscriptJSON(transcript []byte) (int, error) {
	return compact.EstimateMessageTokensFromAPIMessagesJSON(transcript)
}

var (
	tokenBudgetStartRe = regexp.MustCompile(`(?i)^\s*\+(\d+(?:\.\d+)?)\s*(k|m|b)\b`)
	tokenBudgetEndRe   = regexp.MustCompile(`(?i)\s\+(\d+(?:\.\d+)?)\s*(k|m|b)\s*[.!?]?\s*$`)
	tokenBudgetVerbose = regexp.MustCompile(`(?i)\b(?:use|spend)\s+(\d+(?:\.\d+)?)\s*(k|m|b)\s*tokens?\b`)
)

var tokenBudgetMultipliers = map[string]float64{
	"k": 1_000,
	"m": 1_000_000,
	"b": 1_000_000_000,
}

func parseBudgetMatch(value, suffix string) int {
	mul := tokenBudgetMultipliers[strings.ToLower(suffix)]
	if mul == 0 {
		return 0
	}
	f, err := strconv.ParseFloat(value, 64)
	if err != nil || f <= 0 {
		return 0
	}
	n := int(f * mul)
	if n < 0 {
		return 0
	}
	return n
}

// ParseTokenBudget mirrors utils/tokenBudget.ts parseTokenBudget (shorthand + verbose).
func ParseTokenBudget(text string) (budget int, ok bool) {
	if s := tokenBudgetStartRe.FindStringSubmatch(text); len(s) == 3 {
		n := parseBudgetMatch(s[1], s[2])
		if n > 0 {
			return n, true
		}
	}
	if s := tokenBudgetEndRe.FindStringSubmatch(text); len(s) == 3 {
		n := parseBudgetMatch(s[1], s[2])
		if n > 0 {
			return n, true
		}
	}
	if s := tokenBudgetVerbose.FindStringSubmatch(text); len(s) == 3 {
		n := parseBudgetMatch(s[1], s[2])
		if n > 0 {
			return n, true
		}
	}
	return 0, false
}

// BudgetContinuationMessage mirrors getBudgetContinuationMessage (utils/tokenBudget.ts).
func BudgetContinuationMessage(pct, turnTokens, budget int) string {
	return fmt.Sprintf("Stopped at %d%% of token target (%s / %s). Keep working \u2014 do not summarize.",
		pct, formatBudgetInt(turnTokens), formatBudgetInt(budget))
}

func formatBudgetInt(n int) string {
	if n < 0 {
		n = 0
	}
	s := strconv.FormatInt(int64(n), 10)
	var b strings.Builder
	for i, r := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			b.WriteByte(',')
		}
		b.WriteRune(r)
	}
	return b.String()
}

// BudgetTracker mirrors query/tokenBudget.ts BudgetTracker (H5.5).
type BudgetTracker struct {
	ContinuationCount    int
	LastDeltaTokens      int
	LastGlobalTurnTokens int
	StartedAtUnixMilli   int64
}

// NewBudgetTracker mirrors createBudgetTracker.
func NewBudgetTracker() BudgetTracker {
	return BudgetTracker{StartedAtUnixMilli: time.Now().UnixMilli()}
}

const (
	tokenBudgetCompletionThreshold = 0.9
	tokenBudgetDiminishingDelta    = 500
)

// BudgetAction mirrors TokenBudgetDecision branches.
type BudgetAction int

const (
	BudgetActionStop BudgetAction = iota
	BudgetActionContinue
)

// TokenBudgetCompletionEvent mirrors TS completionEvent when action stop with telemetry.
type TokenBudgetCompletionEvent struct {
	ContinuationCount  int
	Pct                int
	TurnTokens         int
	Budget             int
	DiminishingReturns bool
	DurationMs         int64
}

// TokenBudgetDecision mirrors query/tokenBudget.ts checkTokenBudget return (H5.5).
type TokenBudgetDecision struct {
	Action            BudgetAction
	NudgeMessage      string
	ContinuationCount int
	Pct               int
	TurnTokens        int
	Budget            int
	Completion        *TokenBudgetCompletionEvent
}

func budgetPctRounded(turnTokens, budget int) int {
	if budget <= 0 {
		return 0
	}
	return (turnTokens*100 + budget/2) / budget
}

// CheckTokenBudget mirrors query/tokenBudget.ts checkTokenBudget.
func CheckTokenBudget(tracker *BudgetTracker, agentID string, budget int, globalTurnTokens int) TokenBudgetDecision {
	if tracker == nil {
		return TokenBudgetDecision{Action: BudgetActionStop, Completion: nil}
	}
	if strings.TrimSpace(agentID) != "" || budget <= 0 {
		return TokenBudgetDecision{Action: BudgetActionStop, Completion: nil}
	}

	turnTokens := globalTurnTokens
	pct := budgetPctRounded(turnTokens, budget)
	deltaSinceLast := turnTokens - tracker.LastGlobalTurnTokens

	isDiminishing := tracker.ContinuationCount >= 3 &&
		deltaSinceLast < tokenBudgetDiminishingDelta &&
		tracker.LastDeltaTokens < tokenBudgetDiminishingDelta

	if !isDiminishing && float64(turnTokens) < float64(budget)*tokenBudgetCompletionThreshold {
		tracker.ContinuationCount++
		tracker.LastDeltaTokens = deltaSinceLast
		tracker.LastGlobalTurnTokens = turnTokens
		return TokenBudgetDecision{
			Action:            BudgetActionContinue,
			NudgeMessage:      BudgetContinuationMessage(pct, turnTokens, budget),
			ContinuationCount: tracker.ContinuationCount,
			Pct:               pct,
			TurnTokens:        turnTokens,
			Budget:            budget,
		}
	}

	if isDiminishing || tracker.ContinuationCount > 0 {
		dur := time.Now().UnixMilli() - tracker.StartedAtUnixMilli
		return TokenBudgetDecision{
			Action: BudgetActionStop,
			Completion: &TokenBudgetCompletionEvent{
				ContinuationCount:  tracker.ContinuationCount,
				Pct:                pct,
				TurnTokens:         turnTokens,
				Budget:             budget,
				DiminishingReturns: isDiminishing,
				DurationMs:         dur,
			},
		}
	}

	return TokenBudgetDecision{Action: BudgetActionStop, Completion: nil}
}

// EstimateAttachmentRawBytesAsTokens maps raw memdir / inject bytes to heuristic tokens (aligns with coarse 4 bytes/token; H5 / P5.F.1).
func EstimateAttachmentRawBytesAsTokens(rawBytes int) int {
	if rawBytes <= 0 {
		return 0
	}
	return (rawBytes + 3) / 4
}

// EstimateResolvedSubmitTextTokens selects token basis for resolved Submit text when TOKEN_BUDGET max-input-tokens is enforced.
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

// EstimateSubmitTokenBudgetTotal is resolved-text estimate plus attachment pseudo-tokens (H5).
func EstimateSubmitTokenBudgetTotal(mode, resolved string, injectRawBytes int) int {
	if strings.ToLower(strings.TrimSpace(mode)) == "api" {
		return EstimateUTF8BytesAsTokens(resolved) + EstimateAttachmentRawBytesAsTokens(injectRawBytes)
	}
	return EstimateResolvedSubmitTextTokens(mode, resolved) + EstimateAttachmentRawBytesAsTokens(injectRawBytes)
}
