package compact

import (
	"regexp"
	"strings"
)

var (
	reCompactAnalysis = regexp.MustCompile(`(?s)<analysis>.*?</analysis>`)
	reCompactSummary  = regexp.MustCompile(`(?s)<summary>(.*?)</summary>`)
	reMultiNewline    = regexp.MustCompile(`\n\n+`)
)

// FormatCompactSummary mirrors prompt.ts formatCompactSummary (strip <analysis>, unwrap <summary>).
func FormatCompactSummary(summary string) string {
	formatted := reCompactAnalysis.ReplaceAllString(summary, "")
	if m := reCompactSummary.FindStringSubmatch(formatted); len(m) >= 2 {
		content := strings.TrimSpace(m[1])
		formatted = reCompactSummary.ReplaceAllString(formatted, "Summary:\n"+content)
	}
	formatted = reMultiNewline.ReplaceAllString(formatted, "\n\n")
	return strings.TrimSpace(formatted)
}
