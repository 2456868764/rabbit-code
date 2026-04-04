package compact

import (
	"regexp"
	"strconv"
)

// PromptTooLongErrorPrefix mirrors services/api ErrPromptTooLongMessage (errors.ts PROMPT_TOO_LONG_ERROR_MESSAGE).
const PromptTooLongErrorPrefix = "Prompt is too long"

var compactPTLRe = regexp.MustCompile(`(?i)prompt is too long[^0-9]*(\d+)\s*tokens?\s*>\s*(\d+)`)

// ParsePromptTooLongTokenCounts mirrors parsePromptTooLongTokenCounts (errors.ts); duplicated here so compact does not import services/api (breaks api↔compact cycle with AnthropicAssistant).
func ParsePromptTooLongTokenCounts(raw string) (actual, limit int64, ok bool) {
	m := compactPTLRe.FindStringSubmatch(raw)
	if len(m) != 3 {
		return 0, 0, false
	}
	a, _ := strconv.ParseInt(m[1], 10, 64)
	l, _ := strconv.ParseInt(m[2], 10, 64)
	return a, l, true
}
