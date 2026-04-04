package compact

import (
	"strings"
	"testing"
)

func TestFormatCompactSummary_stripsAnalysis(t *testing.T) {
	in := "<analysis>draft</analysis>\n<summary>\nhello\n</summary>"
	got := FormatCompactSummary(in)
	if strings.Contains(got, "analysis") || strings.Contains(got, "draft") {
		t.Fatalf("%q", got)
	}
	if !strings.Contains(got, "Summary:") || !strings.Contains(got, "hello") {
		t.Fatalf("%q", got)
	}
}

func TestFormatCompactSummary_noTags(t *testing.T) {
	in := "  plain text  "
	if g := FormatCompactSummary(in); g != "plain text" {
		t.Fatalf("%q", g)
	}
}

func TestGetCompactPrompt_structure(t *testing.T) {
	p := GetCompactPrompt("")
	if !strings.HasPrefix(p, "CRITICAL: Respond with TEXT ONLY") {
		t.Fatal("preamble")
	}
	if !strings.Contains(p, "Primary Request and Intent") || !strings.Contains(p, "REMINDER: Do NOT call any tools") {
		t.Fatal("body/trailer")
	}
	p2 := GetCompactPrompt("focus on tests")
	if !strings.Contains(p2, "Additional Instructions:\nfocus on tests") {
		t.Fatal(p2)
	}
}

func TestGetPartialCompactPrompt_direction(t *testing.T) {
	from := GetPartialCompactPrompt("", PartialCompactFrom)
	if !strings.Contains(from, "RECENT portion of the conversation") {
		t.Fatal(from)
	}
	up := GetPartialCompactPrompt("", PartialCompactUpTo)
	if !strings.Contains(up, "Context for Continuing Work") || strings.Contains(up, "RECENT portion") {
		t.Fatalf("up_to template: %s", up)
	}
}

func TestGetCompactUserSummaryMessage_paths(t *testing.T) {
	s := GetCompactUserSummaryMessage("<summary>x</summary>", false, "/t.json", true)
	if !strings.Contains(s, "Summary:\nx") || !strings.Contains(s, "/t.json") || !strings.Contains(s, "Recent messages are preserved") {
		t.Fatal(s)
	}
	s2 := GetCompactUserSummaryMessage("hi", true, "", false)
	if !strings.Contains(s2, "Continue the conversation from where it left off") {
		t.Fatal(s2)
	}
}
