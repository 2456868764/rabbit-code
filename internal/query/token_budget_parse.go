package query

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

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
	// U+2014 em dash matches utils/tokenBudget.ts getBudgetContinuationMessage.
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
