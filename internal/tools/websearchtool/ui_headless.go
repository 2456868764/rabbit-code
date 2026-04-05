package websearchtool

import (
	"fmt"
	"math"
	"strings"
)

// ToolUseDescription mirrors async description(input) in WebSearchTool.ts.
func ToolUseDescription(query string) string {
	return fmt.Sprintf("Claude wants to search the web for: %s", strings.TrimSpace(query))
}

// ActivityDescription mirrors getActivityDescription(input) in WebSearchTool.ts.
func ActivityDescription(query string) string {
	s := TruncToolUseSummary(query)
	if s == "" {
		return "Searching the web"
	}
	return "Searching for " + s
}

// TruncToolUseSummary mirrors getToolUseSummary (TOOL_SUMMARY_MAX_LENGTH).
func TruncToolUseSummary(query string) string {
	q := strings.TrimSpace(query)
	if q == "" {
		return ""
	}
	r := []rune(q)
	if len(r) <= ToolSummaryMaxLength {
		return q
	}
	return string(r[:ToolSummaryMaxLength])
}

// SearchCounts mirrors getSearchSummary in UI.tsx (object-shaped results only).
func SearchCounts(results []any) (searchCount, totalHitCount int) {
	for _, r := range results {
		if r == nil {
			continue
		}
		if _, ok := r.(string); ok {
			continue
		}
		if blk, ok := r.(SearchResultBlock); ok {
			searchCount++
			totalHitCount += len(blk.Content)
			continue
		}
		m, ok := r.(map[string]any)
		if !ok {
			continue
		}
		raw, ok := m["content"]
		if !ok || raw == nil {
			searchCount++
			continue
		}
		arr, ok := raw.([]any)
		if !ok {
			searchCount++
			continue
		}
		searchCount++
		totalHitCount += len(arr)
	}
	return searchCount, totalHitCount
}

// FormatToolResultSummaryLine mirrors renderToolResultMessage chrome (text-only headless).
func FormatToolResultSummaryLine(durationSeconds float64, results []any) string {
	n, _ := SearchCounts(results)
	unit := "search"
	if n != 1 {
		unit = "searches"
	}
	var timeDisplay string
	if durationSeconds >= 1 {
		timeDisplay = fmt.Sprintf("%.0fs", math.Round(durationSeconds))
	} else {
		timeDisplay = fmt.Sprintf("%.0fms", math.Round(durationSeconds*1000))
	}
	return fmt.Sprintf("Did %d %s in %s", n, unit, timeDisplay)
}
