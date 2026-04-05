package processuserinput

import (
	"strings"
	"testing"
)

func TestTruncateHookOutput_noop(t *testing.T) {
	s := strings.Repeat("a", 100)
	if got := TruncateHookOutput(s); got != s {
		t.Fatal(got)
	}
}

func TestTruncateHookOutput_truncates(t *testing.T) {
	s := strings.Repeat("b", MaxHookOutputLength+50)
	got := TruncateHookOutput(s)
	if len(got) <= MaxHookOutputLength {
		t.Fatal("expected longer than cap including suffix", len(got))
	}
	if !strings.Contains(got, "truncated") {
		t.Fatal(got)
	}
	if !strings.HasPrefix(got, strings.Repeat("b", MaxHookOutputLength)) {
		t.Fatal("prefix")
	}
}

func TestPlainString(t *testing.T) {
	if PlainString("  x  ") != "x" {
		t.Fatal(PlainString("  x  "))
	}
}
