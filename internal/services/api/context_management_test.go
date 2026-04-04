package anthropic

import (
	"testing"
)

func TestModelSupportsContextManagement_firstParty(t *testing.T) {
	if !ModelSupportsContextManagement("claude-sonnet-4-20250514", ProviderAnthropic) {
		t.Fatal("sonnet 4")
	}
	if ModelSupportsContextManagement("claude-3-5-haiku-20241022", ProviderAnthropic) {
		t.Fatal("claude-3 should be false on 1P")
	}
	if !ModelSupportsContextManagement("", ProviderFoundry) {
		t.Fatal("foundry any model")
	}
}

func TestAppendBetaUnique(t *testing.T) {
	a := []string{"a", "b"}
	got := AppendBetaUnique(a, "a")
	if len(got) != 2 {
		t.Fatal(got)
	}
	got = AppendBetaUnique(a, "c")
	if len(got) != 3 || got[2] != "c" {
		t.Fatal(got)
	}
}
