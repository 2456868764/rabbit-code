package websearchtool

import (
	"strings"
	"testing"

	runewidth "github.com/mattn/go-runewidth"
)

func TestToolUseDescription(t *testing.T) {
	s := ToolUseDescription("  rust async  ")
	want := "Claude wants to search the web for:   rust async  "
	if s != want {
		t.Fatal(s)
	}
}

func TestTruncToolUseSummary(t *testing.T) {
	long := strings.Repeat("a", 60)
	s := TruncToolUseSummary(long)
	if runewidth.StringWidth(s) != ToolSummaryMaxLength {
		t.Fatalf("width got %d want %d: %q", runewidth.StringWidth(s), ToolSummaryMaxLength, s)
	}
	if !strings.HasSuffix(s, "…") {
		t.Fatal(s)
	}
}

func TestRenderToolUseMessage(t *testing.T) {
	if RenderToolUseMessage("", nil, nil, false) != "" {
		t.Fatal()
	}
	got := RenderToolUseMessage(`say "hi"`, []string{"a.com"}, []string{"b.com"}, true)
	if !strings.Contains(got, `say "hi"`) || !strings.Contains(got, "only allowing") || !strings.Contains(got, "blocking domains") {
		t.Fatal(got)
	}
}

func TestUpstreamStringHelpers(t *testing.T) {
	if InnerSearchUserContent("q") != "Perform a web search for the query: q" {
		t.Fatal()
	}
	if AutoClassifierInput(Input{Query: "z"}) != "z" {
		t.Fatal()
	}
	if ExtractSearchText() != "" {
		t.Fatal()
	}
	p := DefaultCheckPermissions()
	if p.Behavior != "passthrough" || p.Message != PermissionMessage || len(p.Suggestions) != 1 {
		t.Fatalf("%+v", p)
	}
	if FormatToolUseProgressHeadless("query_update", "x", 0) != "Searching: x" {
		t.Fatal()
	}
	if FormatToolUseProgressHeadless("search_results_received", "y", 3) != `Found 3 results for "y"` {
		t.Fatal()
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
