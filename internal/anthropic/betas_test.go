package anthropic

import "testing"

func TestMergeBetaHeader(t *testing.T) {
	s := MergeBetaHeader([]string{BetaClaudeCode20250219, "", BetaWebSearch})
	if s != BetaClaudeCode20250219+","+BetaWebSearch {
		t.Fatal(s)
	}
}
