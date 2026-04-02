package anthropic

import (
	"strings"
	"testing"
)

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
