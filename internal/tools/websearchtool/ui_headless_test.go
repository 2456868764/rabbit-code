package websearchtool

import (
	"strings"
	"testing"
)

func TestToolUseDescription(t *testing.T) {
	s := ToolUseDescription("  rust async  ")
	if s != "Claude wants to search the web for: rust async" {
		t.Fatal(s)
	}
}

func TestTruncToolUseSummary(t *testing.T) {
	long := strings.Repeat("あ", 60)
	s := TruncToolUseSummary(long)
	if len([]rune(s)) != ToolSummaryMaxLength {
		t.Fatalf("len %d", len([]rune(s)))
	}
}

func TestFormatToolResultSummaryLine(t *testing.T) {
	results := []any{
		SearchResultBlock{Content: []SearchHit{{}, {}}},
		"note",
		SearchResultBlock{Content: []SearchHit{{}}},
	}
	line := FormatToolResultSummaryLine(2.4, results)
	if line != "Did 2 searches in 2s" {
		t.Fatal(line)
	}
	line = FormatToolResultSummaryLine(0.4, []any{SearchResultBlock{Content: []SearchHit{{}}}})
	if line != "Did 1 search in 400ms" {
		t.Fatal(line)
	}
}
