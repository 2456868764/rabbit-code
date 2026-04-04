package query

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
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

// ImageDocumentTokenEstimate mirrors microCompact.ts IMAGE_MAX_TOKEN_SIZE (2000).
const ImageDocumentTokenEstimate = 2000

// estimateBase64DecodedTokens maps base64 payload length to a coarse token estimate (~1 token / 4 decoded bytes).
func estimateBase64DecodedTokens(b64 string) int {
	b64 = strings.TrimSpace(b64)
	if b64 == "" {
		return 0
	}
	decApprox := (len(b64) * 3) / 4
	if decApprox < 0 {
		return 0
	}
	return (decApprox + 3) / 4
}

// estimateImageOrDocumentBlockTokens uses IMAGE_MAX_TOKEN_SIZE or a larger heuristic when base64 data is present (attachments-style).
func estimateImageOrDocumentBlockTokens(b map[string]json.RawMessage) int {
	n := ImageDocumentTokenEstimate
	srcRaw, ok := b["source"]
	if !ok || len(srcRaw) == 0 {
		return n
	}
	var src map[string]json.RawMessage
	if json.Unmarshal(srcRaw, &src) != nil {
		return n
	}
	data := jsonStringField(src["data"])
	if t := estimateBase64DecodedTokens(data); t > n {
		n = t
	}
	return n
}

// EstimateMessageTokensFromTranscriptJSON mirrors microCompact.ts estimateMessageTokens for API-shaped
// messages JSON ([{role, content}, ...]); pads by ceil(4/3) like TS.
func EstimateMessageTokensFromTranscriptJSON(transcript []byte) (int, error) {
	var arr []map[string]json.RawMessage
	if err := json.Unmarshal(transcript, &arr); err != nil {
		return 0, err
	}
	total := 0
	for _, m := range arr {
		role := jsonStringField(m["role"])
		if role != "user" && role != "assistant" {
			continue
		}
		c := m["content"]
		if len(c) == 0 {
			continue
		}
		switch c[0] {
		case '"':
			var s string
			if err := json.Unmarshal(c, &s); err == nil {
				total += EstimateUTF8BytesAsTokens(s)
			}
		case '[':
			var blocks []map[string]json.RawMessage
			if err := json.Unmarshal(c, &blocks); err != nil {
				continue
			}
			for _, b := range blocks {
				typ := jsonStringField(b["type"])
				switch typ {
				case "text":
					total += EstimateUTF8BytesAsTokens(jsonStringField(b["text"]))
				case "tool_result":
					total += estimateToolResultContentTokens(b["content"])
				case "image", "document":
					total += estimateImageOrDocumentBlockTokens(b)
				case "thinking":
					total += EstimateUTF8BytesAsTokens(jsonStringField(b["thinking"]))
				case "redacted_thinking":
					total += EstimateUTF8BytesAsTokens(jsonStringField(b["data"]))
				case "tool_use":
					name := jsonStringField(b["name"])
					in := ""
					if raw, ok := b["input"]; ok && len(raw) > 0 {
						in = string(raw)
					}
					total += EstimateUTF8BytesAsTokens(name + in)
				default:
					total += EstimateUTF8BytesAsTokens(string(jsonBlockStringify(b)))
				}
			}
		}
	}
	if total == 0 {
		return 0, nil
	}
	return (total*4 + 2) / 3, nil
}

func jsonStringField(raw json.RawMessage) string {
	if len(raw) == 0 || raw[0] != '"' {
		return ""
	}
	var s string
	_ = json.Unmarshal(raw, &s)
	return s
}

func estimateToolResultContentTokens(raw json.RawMessage) int {
	if len(raw) == 0 {
		return 0
	}
	if raw[0] == '"' {
		var s string
		if json.Unmarshal(raw, &s) == nil {
			return EstimateUTF8BytesAsTokens(s)
		}
		return 0
	}
	var arr []map[string]json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return EstimateUTF8BytesAsTokens(string(raw))
	}
	sum := 0
	for _, b := range arr {
		typ := jsonStringField(b["type"])
		switch typ {
		case "text":
			sum += EstimateUTF8BytesAsTokens(jsonStringField(b["text"]))
		case "image", "document":
			sum += estimateImageOrDocumentBlockTokens(b)
		default:
			sum += EstimateUTF8BytesAsTokens(string(jsonBlockStringify(b)))
		}
	}
	return sum
}

func jsonBlockStringify(b map[string]json.RawMessage) string {
	out, err := json.Marshal(b)
	if err != nil {
		return ""
	}
	return string(out)
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
