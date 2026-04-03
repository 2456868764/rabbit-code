package query

import "testing"

func TestEstimateAttachmentRawBytesAsTokens(t *testing.T) {
	if EstimateAttachmentRawBytesAsTokens(0) != 0 {
		t.Fatal()
	}
	if EstimateAttachmentRawBytesAsTokens(8) != 2 {
		t.Fatalf("got %d", EstimateAttachmentRawBytesAsTokens(8))
	}
}

func TestEstimateResolvedSubmitTextTokens_structuredJSON(t *testing.T) {
	// Minimal messages array: one user text block (structured path uses EstimateMessageTokensFromTranscriptJSON).
	j := []byte(`[{"role":"user","content":"abcd"}]`)
	tok := EstimateResolvedSubmitTextTokens("structured", string(j))
	if tok <= 0 {
		t.Fatalf("structured path should return positive tokens, got %d", tok)
	}
	if got := EstimateResolvedSubmitTextTokens("bytes4", string(j)); got != EstimateUTF8BytesAsTokens(string(j)) {
		t.Fatalf("bytes4 mismatch %d vs %d", got, EstimateUTF8BytesAsTokens(string(j)))
	}
}

func TestEstimateSubmitTokenBudgetTotal(t *testing.T) {
	// "abcde" -> 2 tokens; 4 raw bytes -> 1 token; total 3
	if n := EstimateSubmitTokenBudgetTotal("bytes4", "abcde", 4); n != 3 {
		t.Fatalf("got %d", n)
	}
}
