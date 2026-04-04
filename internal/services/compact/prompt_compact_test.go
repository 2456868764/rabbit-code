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
