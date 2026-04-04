package anthropic

import (
	"strings"
	"testing"
)

func TestBetaOAuth_matchesOAuthTS(t *testing.T) {
	if BetaOAuth != "oauth-2025-04-20" {
		t.Fatal(BetaOAuth)
	}
}

func TestBetaCLIInternal_matchesConstantsBetasTS(t *testing.T) {
	if BetaCLIInternal != "cli-internal-2026-02-09" {
		t.Fatal(BetaCLIInternal)
	}
}

func TestDedupeBetasPreserveOrder(t *testing.T) {
	got := DedupeBetasPreserveOrder([]string{
		BetaWebSearch, BetaWebSearch, " " + BetaTaskBudgets, BetaWebSearch,
	})
	if len(got) != 2 || got[0] != BetaWebSearch || got[1] != BetaTaskBudgets {
		t.Fatalf("%v", got)
	}
}

func TestMergeBetasForProvider_dedupesInput(t *testing.T) {
	s := MergeBetasForProvider(ProviderAnthropic, []string{
		BetaClaudeCode20250219, BetaClaudeCode20250219, BetaWebSearch,
	})
	want := BetaClaudeCode20250219 + "," + BetaWebSearch
	if s != want {
		t.Fatalf("got %q want %q", s, want)
	}
}

func TestBedrockExtraParamsBetas_matchesConstantsBetasTS(t *testing.T) {
	want := []string{BetaInterleavedThinking, BetaContext1M, BetaToolSearch3P}
	if len(BedrockExtraParamsBetas) != len(want) {
		t.Fatalf("got %d want %d", len(BedrockExtraParamsBetas), len(want))
	}
	for _, w := range want {
		if _, ok := BedrockExtraParamsBetas[w]; !ok {
			t.Fatalf("missing %q", w)
		}
	}
}

func TestSplitBetasForBedrock(t *testing.T) {
	h, e := SplitBetasForBedrock([]string{
		BetaClaudeCode20250219,
		BetaInterleavedThinking,
		BetaWebSearch,
	})
	if len(h) != 2 || !contains(h, BetaClaudeCode20250219) || !contains(h, BetaWebSearch) {
		t.Fatalf("header %v", h)
	}
	if len(e) != 1 || e[0] != BetaInterleavedThinking {
		t.Fatalf("extra %v", e)
	}
}

func TestMergeBetasForProvider_BedrockOmitsExtra(t *testing.T) {
	s := MergeBetasForProvider(ProviderBedrock, []string{BetaInterleavedThinking, BetaWebSearch})
	if !strings.Contains(s, BetaWebSearch) {
		t.Fatal(s)
	}
	if strings.Contains(s, BetaInterleavedThinking) {
		t.Fatal("interleaved should be extraBody-only for Bedrock", s)
	}
}

func TestMergeBetaHeaderAppend(t *testing.T) {
	s := MergeBetaHeaderAppend(BetaWebSearch, BetaTaskBudgets)
	if s != BetaWebSearch+","+BetaTaskBudgets {
		t.Fatal(s)
	}
	s2 := MergeBetaHeaderAppend(s, BetaTaskBudgets)
	if s2 != s {
		t.Fatal("duplicate append", s2)
	}
}

func TestFilterBetasVertexCountTokens(t *testing.T) {
	out := FilterBetasVertexCountTokens([]string{BetaWebSearch, BetaClaudeCode20250219})
	if len(out) != 1 || out[0] != BetaClaudeCode20250219 {
		t.Fatal(out)
	}
}

func contains(ss []string, x string) bool {
	for _, s := range ss {
		if s == x {
			return true
		}
	}
	return false
}

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
