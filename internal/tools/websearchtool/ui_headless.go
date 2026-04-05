package websearchtool

import (
	"fmt"
	"math"
	"strings"

	runewidth "github.com/mattn/go-runewidth"
	"github.com/rivo/uniseg"
)

// ToolUseDescription mirrors async description(input) in WebSearchTool.ts (raw query, no trim).
func ToolUseDescription(query string) string {
	return fmt.Sprintf("Claude wants to search the web for: %s", query)
}

// ActivityDescription mirrors getActivityDescription(input) in WebSearchTool.ts.
func ActivityDescription(query string) string {
	s := TruncToolUseSummary(query)
	if s == "" {
		return "Searching the web"
	}
	return "Searching for " + s
}

// TruncToolUseSummary mirrors getToolUseSummary + truncate() from utils/truncate.ts (display width, grapheme-safe, ellipsis).
func TruncToolUseSummary(query string) string {
	if query == "" {
		return ""
	}
	if runewidth.StringWidth(query) <= ToolSummaryMaxLength {
		return query
	}
	if ToolSummaryMaxLength <= 1 {
		return "…"
	}
	gr := uniseg.NewGraphemes(query)
	var b strings.Builder
	w := 0
	for gr.Next() {
		seg := gr.Str()
		sw := runewidth.StringWidth(seg)
		if w+sw > ToolSummaryMaxLength-1 {
			break
		}
		b.WriteString(seg)
		w += sw
	}
	return b.String() + "…"
}

// RenderToolUseMessage mirrors UI.tsx renderToolUseMessage (plain-text headless).
func RenderToolUseMessage(query string, allowed, blocked []string, verbose bool) string {
	if query == "" {
		return ""
	}
	msg := `"` + query + `"`
	if verbose {
		if len(allowed) > 0 {
			msg += ", only allowing domains: " + strings.Join(allowed, ", ")
		}
		if len(blocked) > 0 {
			msg += ", blocking domains: " + strings.Join(blocked, ", ")
		}
	}
	return msg
}

// FormatToolUseProgressHeadless mirrors the last renderToolUseProgressMessage line as plain text.
func FormatToolUseProgressHeadless(dataType, query string, resultCount int) string {
	switch dataType {
	case "query_update":
		return "Searching: " + query
	case "search_results_received":
		return fmt.Sprintf("Found %d results for %q", resultCount, query)
	default:
		return ""
	}
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
